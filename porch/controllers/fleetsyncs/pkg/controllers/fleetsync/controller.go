// Copyright 2023 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fleetsync

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/pkg/controllers/fleetsync/fleetpoller"
	gkehubv1 "google.golang.org/api/gkehub/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	fleetSyncLabel = "config.porch.kpt.dev/fleetsync"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

func NewFleetSyncReconciler() *FleetSyncReconciler {
	return &FleetSyncReconciler{}
}

type FleetSyncReconciler struct {
	Options

	client.Client

	poller *fleetpoller.Poller
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 rbac:roleName=porch-controllers-fleetsyncs webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/finalizers,verbs=update

func (r *FleetSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var fleetsync v1alpha1.FleetSync
	if err := r.Get(ctx, req.NamespacedName, &fleetsync); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	org := fleetsync.DeepCopy()

	myFinalizerName := "config.porch.kpt.dev/fleetsyncs"
	if fleetsync.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&fleetsync, myFinalizerName) {
			controllerutil.AddFinalizer(&fleetsync, myFinalizerName)
			if err := r.Update(ctx, &fleetsync); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&fleetsync, myFinalizerName) {
			// remove our finalizer from the list and update it.
			r.poller.StopPollingForFleetSync(req.NamespacedName)
			controllerutil.RemoveFinalizer(&fleetsync, myFinalizerName)
			if err := r.Update(ctx, &fleetsync); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	klog.Infof("Reconciling %s", req.NamespacedName.String())

	r.poller.VerifyProjectIdsForFleetSync(req.NamespacedName, fleetsync.Spec.ProjectIds)

	allFound := true
	errors := make(map[string]error)
	var hubMemberships []*gkehubv1.Membership
	for _, projectId := range fleetsync.Spec.ProjectIds {
		memberships, err, found := r.poller.LatestPollResult(projectId)
		if !found {
			allFound = false
		}
		if err != nil {
			errors[projectId] = err
			continue
		}
		hubMemberships = append(hubMemberships, memberships...)
	}

	if !allFound {
		meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: fleetsync.Generation,
			Reason:             "NotSynced",
		})
		meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
			Type:               "Stalled",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: fleetsync.Generation,
			Reason:             "Progressing",
		})
		if err := r.updateStatus(ctx, org, &fleetsync); err != nil {
			klog.Infof("Error updating status for %s: %v", req.NamespacedName.String(), err)
		}
		return ctrl.Result{}, nil
	}

	if len(errors) != 0 {
		var builder strings.Builder
		builder.WriteString("Errors fetching memberships in fleet:\n")
		for projectId, err := range errors {
			builder.WriteString(fmt.Sprintf("%s: %s\n", projectId, err.Error()))
		}
		meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			ObservedGeneration: fleetsync.Generation,
			Reason:             "FleetSyncError",
		})
		meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
			Type:               "Stalled",
			Status:             metav1.ConditionTrue,
			ObservedGeneration: fleetsync.Generation,
			Reason:             "FleetSyncError",
			Message:            builder.String(),
		})
		if err := r.updateStatus(ctx, org, &fleetsync); err != nil {
			klog.Infof("Error updating status for %s: %v", req.NamespacedName.String(), err)
		}
		return ctrl.Result{}, nil
	}

	existingMemberships, err := r.findExistingMemberships(ctx, fleetsync.Name, fleetsync.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, hubMembership := range hubMemberships {
		m := newMembership(hubMembership, &fleetsync)
		existingMembership, found := findMembership(hubMembership, existingMemberships)
		if !found {
			// TODO: We should probably use SSA here rather than Create/Update.
			if err := r.Create(ctx, m); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}

		if !equality.Semantic.DeepEqual(m.Spec, existingMembership.Spec) {
			existingMembership.Spec = m.Spec
			if err := r.Update(ctx, existingMembership); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	for _, m := range existingMemberships {
		name := m.GetName()
		found := false
		for _, hubm := range hubMemberships {
			hubmName := membershipId(hubm)
			if hubmName == name {
				found = true
			}
		}
		if !found {
			if err := r.Delete(ctx, m); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fleetsync.Generation,
		Reason:             "Synced",
	})
	meta.SetStatusCondition(&fleetsync.Status.Conditions, metav1.Condition{
		Type:               "Stalled",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: fleetsync.Generation,
		Reason:             "Synced",
	})
	if err := r.updateStatus(ctx, org, &fleetsync); err != nil {
		klog.Infof("Error updating status for %s: %v", req.NamespacedName.String(), err)
	}
	return ctrl.Result{}, nil
}

func findMembership(membership *gkehubv1.Membership, existing []*v1alpha1.FleetMembership) (*v1alpha1.FleetMembership, bool) {
	name := membershipId(membership)
	for i := range existing {
		em := existing[i]
		if em.Name == name {
			return em, true
		}
	}
	return nil, false
}

func newMembership(hubMembership *gkehubv1.Membership, fleetsync *v1alpha1.FleetSync) *v1alpha1.FleetMembership {
	t := true
	return &v1alpha1.FleetMembership{
		ObjectMeta: metav1.ObjectMeta{
			Name:      membershipId(hubMembership),
			Namespace: fleetsync.Namespace,
			Labels: map[string]string{
				fleetSyncLabel: fleetsync.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: fleetsync.APIVersion,
					Kind:       fleetsync.Kind,
					Name:       fleetsync.Name,
					UID:        fleetsync.UID,
					Controller: &t,
				},
			},
		},
		Spec: v1alpha1.FleetMembershipSpec{
			Description: hubMembership.Description,
			Labels:      hubMembership.Labels,
			State: v1alpha1.MembershipState{
				Code: toMembershipStateCode(hubMembership.State),
			},
		},
	}
}

func toMembershipStateCode(ms *gkehubv1.MembershipState) v1alpha1.MembershipStateCode {
	if ms == nil {
		return v1alpha1.CodeUnspecified
	}

	switch ms.Code {
	case "CODE_UNSPECIFIED":
		return v1alpha1.CodeUnspecified
	case "CREATING":
		return v1alpha1.CodeCreating
	case "READY":
		return v1alpha1.CodeReady
	case "DELETING":
		return v1alpha1.CodeDeleting
	case "UPDATING":
		return v1alpha1.CodeUpdating
	case "SERVICE_UPDATING":
		return v1alpha1.CodeServiceUpdating
	default:
		return v1alpha1.CodeUnspecified
	}
}

func membershipId(hubMembership *gkehubv1.Membership) string {
	segments := strings.Split(hubMembership.Name, "/")
	return segments[(len(segments) - 1)]
}

func (r *FleetSyncReconciler) updateStatus(ctx context.Context, org, new *v1alpha1.FleetSync) error {
	if equality.Semantic.DeepEqual(org.Status, new.Status) {
		return nil
	}
	return r.Status().Update(ctx, new)
}

func (r *FleetSyncReconciler) findExistingMemberships(ctx context.Context, fsName, fsNamespace string) ([]*v1alpha1.FleetMembership, error) {
	var list v1alpha1.FleetMembershipList
	if err := r.List(ctx, &list, client.MatchingLabels{fleetSyncLabel: fsName}, client.InNamespace(fsNamespace)); err != nil {
		return nil, err
	}

	var memberships []*v1alpha1.FleetMembership
	for i := range list.Items {
		item := &list.Items[i]
		memberships = append(memberships, item)
	}
	return memberships, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FleetSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()

	channel := make(chan event.GenericEvent)
	r.poller = fleetpoller.NewPoller(channel)
	r.poller.Start()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FleetSync{}).
		Owns(&v1alpha1.FleetMembership{}).
		Watches(&source.Channel{Source: channel}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
