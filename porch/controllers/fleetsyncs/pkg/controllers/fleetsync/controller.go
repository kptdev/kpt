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

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetmemberships,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetmembershipbindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetscopes,verbs=get;list;watch;create;update;patch;delete

func (r *FleetSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var fleetsync v1alpha1.FleetSync
	if err := r.Get(ctx, req.NamespacedName, &fleetsync); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	orig := fleetsync.DeepCopy()

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

	return r.reconcileMemberships(ctx, req, orig, &fleetsync)
}

func (r *FleetSyncReconciler) reconcileMemberships(ctx context.Context, req ctrl.Request, orig, fleetsync *v1alpha1.FleetSync) (ctrl.Result, error) {
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
		r.setProgressingCondition(ctx, orig, fleetsync)
		return ctrl.Result{}, nil
	}

	if len(errors) != 0 {
		r.setErrorCondition(ctx, orig, fleetsync, "memberships", errors)
		return ctrl.Result{}, nil
	}

	existingMemberships, err := r.findExistingMemberships(ctx, fleetsync.Name, fleetsync.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, hubMembership := range hubMemberships {
		m, err := newMembership(hubMembership, fleetsync)
		if err != nil {
			klog.Warningf("could not create new membership: %s", err.Error())
			continue
		}
		existingMembership, found := findMembership(hubMembership, existingMemberships)
		if !found {
			// TODO: We should probably use SSA here rather than Create/Update.
			if err := r.Create(ctx, m); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}

		if !equality.Semantic.DeepEqual(m.Data, existingMembership.Data) {
			existingMembership.Data = m.Data
			if err := r.Update(ctx, existingMembership); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	for _, m := range existingMemberships {
		name := m.GetName()
		found := false
		for _, hubm := range hubMemberships {
			hubmName, err := membershipId(hubm)
			if err != nil {
				klog.Warning(err)
				continue
			}
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

	r.setReadyCondition(ctx, orig, fleetsync)
	return ctrl.Result{}, nil
}

func (r *FleetSyncReconciler) setReadyCondition(ctx context.Context, orig, fleetsync *v1alpha1.FleetSync) {
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
	if err := r.updateStatus(ctx, orig, fleetsync); err != nil {
		klog.Errorf("Error updating status for %s/%s: %v", fleetsync.Namespace, fleetsync.Name, err)
	}
}

func (r *FleetSyncReconciler) setProgressingCondition(ctx context.Context, orig, fleetsync *v1alpha1.FleetSync) {
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
	if err := r.updateStatus(ctx, orig, fleetsync); err != nil {
		klog.Errorf("Error updating status for %s/%s: %v", fleetsync.Namespace, fleetsync.Name, err)
	}
}

func (r *FleetSyncReconciler) setErrorCondition(ctx context.Context, orig, fleetsync *v1alpha1.FleetSync, resource string, errors map[string]error) {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Errors fetching %s in fleet:\n", resource))
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
	if err := r.updateStatus(ctx, orig, fleetsync); err != nil {
		klog.Errorf("Error updating status for %s/%s: %v", fleetsync.Namespace, fleetsync.Name, err)
	}
}

func findMembership(membership *gkehubv1.Membership, existing []*v1alpha1.FleetMembership) (*v1alpha1.FleetMembership, bool) {
	name, err := membershipId(membership)
	if err != nil {
		klog.Warning(err)
		return nil, false
	}
	for i := range existing {
		em := existing[i]
		if em.Name == name {
			return em, true
		}
	}
	return nil, false
}

func newMembership(hubMembership *gkehubv1.Membership, fleetsync *v1alpha1.FleetSync) (*v1alpha1.FleetMembership, error) {
	t := true

	id, err := membershipId(hubMembership)
	if err != nil {
		return nil, err
	}

	segments := strings.Split(hubMembership.Name, "/")
	if len(segments) != 6 {
		return nil, fmt.Errorf("invalid membership name %q; should be 6 segments")
	}

	return &v1alpha1.FleetMembership{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
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
		Data: v1alpha1.FleetMembershipData{
			FullName:    hubMembership.Name,
			Project:     segments[1],
			Location:    segments[3],
			Membership:  segments[5],
			Description: hubMembership.Description,
			Labels:      hubMembership.Labels,
			State: v1alpha1.MembershipState{
				Code: toMembershipStateCode(hubMembership.State),
			},
		},
	}, nil
}

func toMembershipStateCode(ms *gkehubv1.MembershipState) v1alpha1.MembershipStateCode {
	if ms == nil {
		return v1alpha1.MSCodeUnspecified
	}

	switch ms.Code {
	case "CODE_UNSPECIFIED":
		return v1alpha1.MSCodeUnspecified
	case "CREATING":
		return v1alpha1.MSCodeCreating
	case "READY":
		return v1alpha1.MSCodeReady
	case "DELETING":
		return v1alpha1.MSCodeDeleting
	case "UPDATING":
		return v1alpha1.MSCodeUpdating
	case "SERVICE_UPDATING":
		return v1alpha1.MSCodeServiceUpdating
	default:
		return v1alpha1.MSCodeUnspecified
	}
}

func membershipId(hubMembership *gkehubv1.Membership) (string, error) {
	// projects/*/locations/*/memberships/{membership_id}
	segments := strings.Split(hubMembership.Name, "/")
	if len(segments) != 6 {
		return "", fmt.Errorf("invalid membership name %q; should be 6 segments", hubMembership.Name)
	}
	return segments[1] + segments[3] + segments[5], nil
}

func (r *FleetSyncReconciler) updateStatus(ctx context.Context, orig, new *v1alpha1.FleetSync) error {
	if equality.Semantic.DeepEqual(orig.Status, new.Status) {
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
