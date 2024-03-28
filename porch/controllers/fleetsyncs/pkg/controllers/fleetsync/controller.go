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

	"k8s.io/client-go/tools/record"

	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/pkg/controllers/fleetsync/fleetpoller"
	"github.com/GoogleContainerTools/kpt/porch/pkg/util"
	gkehubv1 "google.golang.org/api/gkehub/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"

	corev1 "k8s.io/api/core/v1"
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
	fleetSyncLabel  = "fleetsync.porch.kpt.dev/fleetsync"
	projectLabel    = "fleetsync.porch.kpt.dev/project"
	scopeLabel      = "fleetsync.porch.kpt.dev/scope"
	membershipLabel = "fleetsync.porch.kpt.dev/membership"
	nameMaxLen      = 63
	nameHashLen     = 8
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

	poller   *fleetpoller.Poller
	recorder record.EventRecorder
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-fleetsyncs webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetsyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetmemberships,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetmembershipbindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=fleetscopes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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

	allFailed := true
	for _, projectId := range fleetsync.Spec.ProjectIds {
		err := r.reconcileProject(ctx, projectId, orig, &fleetsync)
		if err != nil {
			r.recorder.Event(&fleetsync, corev1.EventTypeWarning, "ProjectSyncError",
				fmt.Sprintf("could not sync project %q: %s", projectId, err.Error()))
		} else {
			allFailed = false
		}
	}
	if allFailed {
		r.setErrorCondition(ctx, orig, &fleetsync, "No projects succesfully reconciled")
	}

	return ctrl.Result{}, nil
}

func (r *FleetSyncReconciler) reconcileProject(ctx context.Context, projectId string, orig, fleetsync *v1alpha1.FleetSync) error {
	pr, found := r.poller.LatestResult(projectId)
	if !found {
		r.recorder.Event(fleetsync, corev1.EventTypeNormal, "ProjectSyncPending",
			fmt.Sprintf("Waiting for sync for project %q", projectId))
		return nil
	}

	// If there are any errors for this project ID, we will not
	// sync any data for the project.
	if pr.HasError() {
		return pr.ErrorSummary()
	}

	err := r.reconcileMemberships(ctx, projectId, pr, fleetsync)
	if err != nil {
		return err
	}

	err = r.reconcileScopes(ctx, projectId, pr, fleetsync)
	if err != nil {
		return err
	}

	err = r.reconcileMembershipBindings(ctx, projectId, pr, fleetsync)
	if err != nil {
		return err
	}

	r.setReadyCondition(ctx, orig, fleetsync)
	return nil
}

func (r *FleetSyncReconciler) reconcileMemberships(ctx context.Context, projectId string, pr *fleetpoller.PollResult, fleetsync *v1alpha1.FleetSync) error {
	existingMemberships, err := r.findExistingMemberships(ctx, fleetsync.Name, fleetsync.Namespace, projectId)
	if err != nil {
		return err
	}

	for _, hubm := range pr.Memberships {
		name, err := membershipId(hubm)
		if err != nil {
			klog.Warningf("could not create new membership: %s", err.Error())
			continue
		}

		existing, found := existingMemberships[name]
		if !found {
			m, err := newMembership(hubm, fleetsync)
			if err != nil {
				klog.Warningf("could not create new membership: %s", err.Error())
				continue
			}
			// TODO: We should probably use SSA here rather than Create/Update.
			if err := r.Create(ctx, m); err != nil {
				return err
			}
			continue
		}

		updated := existing.DeepCopy()
		err = updateMembership(hubm, fleetsync, updated)
		if err != nil {
			klog.Warningf("could not update membership: %s", err.Error())
			continue
		}

		if !equality.Semantic.DeepEqual(updated.Data, existing.Data) {
			if err := r.Update(ctx, updated); err != nil {
				return err
			}
		}
	}

	for name, m := range existingMemberships {
		found := false
		for _, hubm := range pr.Memberships {
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
				return err
			}
		}
	}

	return nil
}

func (r *FleetSyncReconciler) reconcileScopes(ctx context.Context, projectId string, pr *fleetpoller.PollResult, fleetsync *v1alpha1.FleetSync) error {
	existingScopes, err := r.findExistingScopes(ctx, fleetsync.Name, fleetsync.Namespace, projectId)
	if err != nil {
		return err
	}

	for _, res := range pr.Scopes {
		name, err := scopeId(res)
		if err != nil {
			klog.Warningf("could not create new scope: %s", err.Error())
			continue
		}

		existing, found := existingScopes[name]
		if !found {
			m, err := newScope(res, fleetsync)
			if err != nil {
				klog.Warningf("could not create new scope: %s", err.Error())
				continue
			}
			// TODO: We should probably use SSA here rather than Create/Update.
			if err := r.Create(ctx, m); err != nil {
				return err
			}
			continue
		}

		updated := existing.DeepCopy()
		err = updateScope(res, fleetsync, updated)
		if err != nil {
			klog.Warningf("could not update scope: %s", err.Error())
			continue
		}

		if !equality.Semantic.DeepEqual(updated.Data, existing.Data) {
			if err := r.Update(ctx, updated); err != nil {
				return err
			}
		}
	}

	for name, m := range existingScopes {
		found := false
		for _, res := range pr.Scopes {
			resName, err := scopeId(res)
			if err != nil {
				klog.Warning(err)
				continue
			}
			if resName == name {
				found = true
			}
		}
		if !found {
			if err := r.Delete(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *FleetSyncReconciler) reconcileMembershipBindings(ctx context.Context, projectId string, pr *fleetpoller.PollResult, fleetsync *v1alpha1.FleetSync) error {
	existingBindings, err := r.findExistingMembershipBindings(ctx, fleetsync.Name, fleetsync.Namespace, projectId)
	if err != nil {
		return err
	}

	for _, res := range pr.Bindings {
		name, err := bindingId(res)
		if err != nil {
			klog.Warningf("could not create new binding: %s", err.Error())
			continue
		}

		existing, found := existingBindings[name]
		if !found {
			m, err := newMembershipBinding(res, fleetsync)
			if err != nil {
				klog.Warningf("could not create new binding: %s", err.Error())
				continue
			}
			// TODO: We should probably use SSA here rather than Create/Update.
			if err := r.Create(ctx, m); err != nil {
				return err
			}
			continue
		}

		updated := existing.DeepCopy()
		err = updateMembershipBinding(res, fleetsync, updated)
		if err != nil {
			klog.Warningf("could not update binding: %s", err.Error())
			continue
		}

		if !equality.Semantic.DeepEqual(updated.Data, existing.Data) {
			if err := r.Update(ctx, updated); err != nil {
				return err
			}
		}
	}

	for name, m := range existingBindings {
		found := false
		for _, res := range pr.Bindings {
			resName, err := bindingId(res)
			if err != nil {
				klog.Warning(err)
				continue
			}
			if resName == name {
				found = true
			}
		}
		if !found {
			if err := r.Delete(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
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

func (r *FleetSyncReconciler) setErrorCondition(ctx context.Context, orig, fleetsync *v1alpha1.FleetSync, message string) {
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
		Message:            message,
	})
	if err := r.updateStatus(ctx, orig, fleetsync); err != nil {
		klog.Errorf("Error updating status for %s/%s: %v", fleetsync.Namespace, fleetsync.Name, err)
	}
}

func newMembership(hubMembership *gkehubv1.Membership, fleetsync *v1alpha1.FleetSync) (*v1alpha1.FleetMembership, error) {
	id, err := membershipId(hubMembership)
	if err != nil {
		return nil, err
	}

	t := true
	fm := &v1alpha1.FleetMembership{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: fleetsync.Namespace,
			Labels:    map[string]string{},
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
	}

	return fm, updateMembership(hubMembership, fleetsync, fm)
}

func updateMembership(hubMembership *gkehubv1.Membership, fleetsync *v1alpha1.FleetSync, fm *v1alpha1.FleetMembership) error {
	segments := strings.Split(hubMembership.Name, "/")
	if len(segments) != 6 {
		return fmt.Errorf("invalid membership name %q; should be 6 segments", hubMembership.Name)
	}

	fm.ObjectMeta.Labels[fleetSyncLabel] = fleetsync.Name
	fm.ObjectMeta.Labels[projectLabel] = segments[1]
	fm.ObjectMeta.Labels[membershipLabel] = segments[5]

	fm.Data = v1alpha1.FleetMembershipData{
		FullName:    hubMembership.Name,
		Project:     segments[1],
		Location:    segments[3],
		Membership:  segments[5],
		Description: hubMembership.Description,
		Labels:      hubMembership.Labels,
		State: v1alpha1.MembershipState{
			Code: toMembershipStateCode(hubMembership.State),
		},
	}

	return nil
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
	return util.KubernetesName(segments[1]+"-"+segments[3]+"-"+segments[5], nameHashLen, nameMaxLen), nil
}

func newScope(hubScope *gkehubv1.Scope, fleetsync *v1alpha1.FleetSync) (*v1alpha1.FleetScope, error) {
	id, err := scopeId(hubScope)
	if err != nil {
		return nil, err
	}

	t := true
	f := &v1alpha1.FleetScope{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: fleetsync.Namespace,
			Labels:    map[string]string{},
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
	}

	return f, updateScope(hubScope, fleetsync, f)
}

func updateScope(hubScope *gkehubv1.Scope, fleetsync *v1alpha1.FleetSync, f *v1alpha1.FleetScope) error {
	segments := strings.Split(hubScope.Name, "/")
	if len(segments) != 6 {
		return fmt.Errorf("invalid scope name %q; should be 6 segments", hubScope.Name)
	}

	f.ObjectMeta.Labels[fleetSyncLabel] = fleetsync.Name
	f.ObjectMeta.Labels[projectLabel] = segments[1]
	f.ObjectMeta.Labels[scopeLabel] = segments[5]

	f.Data = v1alpha1.FleetScopeData{
		FullName: hubScope.Name,
		Project:  segments[1],
		Location: segments[3],
		Scope:    segments[5],
		Labels:   hubScope.Labels,
		State: v1alpha1.ScopeState{
			Code: toScopeStateCode(hubScope.State),
		},
	}

	return nil
}

func toScopeStateCode(ms *gkehubv1.ScopeLifecycleState) v1alpha1.ScopeStateCode {
	if ms == nil {
		return v1alpha1.SSCodeUnspecified
	}

	switch ms.Code {
	case "CODE_UNSPECIFIED":
		return v1alpha1.SSCodeUnspecified
	case "CREATING":
		return v1alpha1.SSCodeCreating
	case "READY":
		return v1alpha1.SSCodeReady
	case "DELETING":
		return v1alpha1.SSCodeDeleting
	case "UPDATING":
		return v1alpha1.SSCodeUpdating
	default:
		return v1alpha1.SSCodeUnspecified
	}
}

func scopeId(scope *gkehubv1.Scope) (string, error) {
	// `projects/{project}/locations/{location}/scopes/{scope}`
	segments := strings.Split(scope.Name, "/")
	if len(segments) != 6 {
		return "", fmt.Errorf("invalid scope name %q; should be 6 segments", scope.Name)
	}
	return util.KubernetesName(segments[1]+"-"+segments[3]+"-"+segments[5], nameHashLen, nameMaxLen), nil
}

func newMembershipBinding(hubBinding *gkehubv1.MembershipBinding, fleetsync *v1alpha1.FleetSync) (*v1alpha1.FleetMembershipBinding, error) {
	id, err := bindingId(hubBinding)
	if err != nil {
		return nil, err
	}

	t := true
	f := &v1alpha1.FleetMembershipBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      id,
			Namespace: fleetsync.Namespace,
			Labels:    map[string]string{},
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
	}

	return f, updateMembershipBinding(hubBinding, fleetsync, f)
}

func updateMembershipBinding(hubBinding *gkehubv1.MembershipBinding, fleetsync *v1alpha1.FleetSync, f *v1alpha1.FleetMembershipBinding) error {
	segments := strings.Split(hubBinding.Name, "/")
	if len(segments) != 8 {
		return fmt.Errorf("invalid binding name %q; should be 8 segments", hubBinding.Name)
	}

	scopeSegs := strings.Split(hubBinding.Scope, "/")
	if len(scopeSegs) != 6 {
		return fmt.Errorf("invalid scope name %q; should be 6 segments", hubBinding.Scope)
	}

	f.ObjectMeta.Labels[fleetSyncLabel] = fleetsync.Name
	f.ObjectMeta.Labels[projectLabel] = segments[1]
	f.ObjectMeta.Labels[membershipLabel] = segments[5]
	f.ObjectMeta.Labels[scopeLabel] = scopeSegs[5]

	f.Data = v1alpha1.FleetMembershipBindingData{
		FullName:   hubBinding.Name,
		Project:    segments[1],
		Location:   segments[3],
		Membership: segments[5],
		Binding:    segments[7],
		Scope:      hubBinding.Scope,
		Labels:     hubBinding.Labels,
		State: v1alpha1.MembershipBindingState{
			Code: toMembershipBindingStateCode(hubBinding.State),
		},
	}

	return nil
}

func toMembershipBindingStateCode(ms *gkehubv1.MembershipBindingLifecycleState) v1alpha1.MembershipBindingStateCode {
	if ms == nil {
		return v1alpha1.MBSCodeUnspecified
	}

	switch ms.Code {
	case "CODE_UNSPECIFIED":
		return v1alpha1.MBSCodeUnspecified
	case "CREATING":
		return v1alpha1.MBSCodeCreating
	case "READY":
		return v1alpha1.MBSCodeReady
	case "DELETING":
		return v1alpha1.MBSCodeDeleting
	case "UPDATING":
		return v1alpha1.MBSCodeUpdating
	default:
		return v1alpha1.MBSCodeUnspecified
	}
}

func bindingId(binding *gkehubv1.MembershipBinding) (string, error) {
	// `projects/{project}/locations/{location}/memberships/{membership}/bindings/{membershipbinding}`
	segments := strings.Split(binding.Name, "/")
	if len(segments) != 8 {
		return "", fmt.Errorf("invalid membership binding name %q; should be 8 segments", binding.Name)
	}
	return util.KubernetesName(segments[1]+"-"+segments[3]+"-"+segments[5]+"-"+segments[7], nameHashLen, nameMaxLen), nil
}

func (r *FleetSyncReconciler) updateStatus(ctx context.Context, orig, new *v1alpha1.FleetSync) error {
	if equality.Semantic.DeepEqual(orig.Status, new.Status) {
		return nil
	}
	return r.Status().Update(ctx, new)
}

func (r *FleetSyncReconciler) findExistingMemberships(ctx context.Context, fsName, fsNamespace, projectId string) (map[string]*v1alpha1.FleetMembership, error) {
	var list v1alpha1.FleetMembershipList
	if err := r.List(ctx, &list, client.MatchingLabels{fleetSyncLabel: fsName, projectLabel: projectId}, client.InNamespace(fsNamespace)); err != nil {
		return nil, err
	}

	items := make(map[string]*v1alpha1.FleetMembership, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		items[item.Name] = item
	}
	return items, nil
}

func (r *FleetSyncReconciler) findExistingScopes(ctx context.Context, fsName, fsNamespace, projectId string) (map[string]*v1alpha1.FleetScope, error) {
	var list v1alpha1.FleetScopeList
	if err := r.List(ctx, &list, client.MatchingLabels{fleetSyncLabel: fsName, projectLabel: projectId}, client.InNamespace(fsNamespace)); err != nil {
		return nil, err
	}

	items := make(map[string]*v1alpha1.FleetScope, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		items[item.Name] = item
	}
	return items, nil
}

func (r *FleetSyncReconciler) findExistingMembershipBindings(ctx context.Context, fsName, fsNamespace, projectId string) (map[string]*v1alpha1.FleetMembershipBinding, error) {
	var list v1alpha1.FleetMembershipBindingList
	if err := r.List(ctx, &list, client.MatchingLabels{fleetSyncLabel: fsName, projectLabel: projectId}, client.InNamespace(fsNamespace)); err != nil {
		return nil, err
	}

	items := make(map[string]*v1alpha1.FleetMembershipBinding, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		items[item.Name] = item
	}
	return items, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FleetSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()
	r.recorder = mgr.GetEventRecorderFor("fleetsync-controller")

	channel := make(chan event.GenericEvent)
	r.poller = fleetpoller.NewPoller(channel)
	r.poller.Start()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FleetSync{}).
		Owns(&v1alpha1.FleetMembership{}).
		Owns(&v1alpha1.FleetScope{}).
		Owns(&v1alpha1.FleetMembershipBinding{}).
		Watches(&source.Channel{Source: channel}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
