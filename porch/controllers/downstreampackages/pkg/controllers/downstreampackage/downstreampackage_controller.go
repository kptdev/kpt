// Copyright 2022 Google LLC
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

package downstreampackage

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/downstreampackages/api/v1alpha1"
	"golang.org/x/mod/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
)

type Options struct{}

func (o *Options) InitDefaults()                       {}
func (o *Options) BindFlags(_ string, _ *flag.FlagSet) {}

// DownstreamPackageReconciler reconciles a DownstreamPackage object
type DownstreamPackageReconciler struct {
	client.Client
	Options
}

const (
	workspaceNamePrefix = "downstreampackage-"
)

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 rbac:roleName=porch-controllers-downstreampackages webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=downstreampackages,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=downstreampackages/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=downstreampackages/finalizers,verbs=update

// Reconcile implements the main kubernetes reconciliation loop.
func (r *DownstreamPackageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dp, prList, err := r.init(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	if dp == nil {
		// maybe the dp was deleted
		return ctrl.Result{}, nil
	}

	upstream := r.getUpstreamPR(dp.Spec.Upstream, prList)
	if upstream == nil {
		return ctrl.Result{}, fmt.Errorf("could not find upstream package revision")
	}

	if err := r.ensureDownstreamPackage(ctx, dp, upstream, prList); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: Prune (propose deletion of) deployment packages created by this controller
	//   that are no longer needed. Part of this will be to implement the DeletionPolicy
	//   field, which will allow the user to specify whether to "orphan" or "delete".

	return ctrl.Result{}, nil
}

func (r *DownstreamPackageReconciler) init(ctx context.Context,
	req ctrl.Request) (*api.DownstreamPackage, *porchapi.PackageRevisionList, error) {
	var dp api.DownstreamPackage
	if err := r.Client.Get(ctx, req.NamespacedName, &dp); err != nil {
		return nil, nil, client.IgnoreNotFound(err)
	}

	var prList porchapi.PackageRevisionList
	if err := r.Client.List(ctx, &prList, client.InNamespace(dp.Namespace)); err != nil {
		return nil, nil, err
	}

	return &dp, &prList, nil
}

func (r *DownstreamPackageReconciler) getUpstreamPR(upstream *api.Upstream,
	prList *porchapi.PackageRevisionList) *porchapi.PackageRevision {
	for _, pr := range prList.Items {
		if pr.Spec.RepositoryName == upstream.Repo &&
			pr.Spec.PackageName == upstream.Package &&
			pr.Spec.Revision == upstream.Revision {
			return &pr
		}
	}
	return nil
}

// ensureDownstreamPackage needs to:
//   - Check if the downstream package revision already exists. If not, create it.
//   - If it does already exist, we need to make sure it is up-to-date. If there is
//     a downstream package draft, we look at the draft. Otherwise, we look at the latest
//     published downstream package revision.
//   - Compare pd.Spec.Upstream.Revision to the revision number that the downstream
//     package is based on. If it is different, we need to do an update (could be an upgrade
//     or a downgrade).
func (r *DownstreamPackageReconciler) ensureDownstreamPackage(ctx context.Context,
	dp *api.DownstreamPackage,
	upstream *porchapi.PackageRevision,
	prList *porchapi.PackageRevisionList) error {
	existing, err := r.findAndUpdateExistingRevision(ctx, dp, upstream, prList)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	// No downstream package created by this controller exists. Create one.
	tr := true
	newPR := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: dp.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         dp.APIVersion,
				Kind:               dp.Kind,
				Name:               dp.Name,
				UID:                dp.UID,
				Controller:         &tr,
				BlockOwnerDeletion: nil,
			}},
		},
		Spec: porchapi.PackageRevisionSpec{
			PackageName:    dp.Spec.Downstream.Package,
			WorkspaceName:  porchapi.WorkspaceName(workspaceNamePrefix + "1"),
			RepositoryName: dp.Spec.Downstream.Repo,
			Tasks: []porchapi.Task{
				{
					Type: porchapi.TaskTypeClone,
					Clone: &porchapi.PackageCloneTaskSpec{
						Upstream: porchapi.UpstreamPackage{
							UpstreamRef: &porchapi.PackageRevisionRef{
								Name: upstream.Name,
							},
						},
					},
				},
			},
		},
	}

	if err := r.Client.Create(ctx, newPR); err != nil {
		return err
	}

	return nil
}

func (r *DownstreamPackageReconciler) findAndUpdateExistingRevision(ctx context.Context,
	dp *api.DownstreamPackage,
	upstream *porchapi.PackageRevision,
	prList *porchapi.PackageRevisionList) (*porchapi.PackageRevision, error) {
	// First, check if a downstream package exists. If not, just return nil. The
	// caller will create one.
	downstream := r.getDownstreamPR(dp, prList)
	if downstream == nil {
		return nil, nil
	}

	// Determine if the downstream package needs to be updated. If not, return
	// the downstream package as-is.
	if r.isUpToDate(dp, downstream) {
		return downstream, nil
	}

	if porchapi.LifecycleIsPublished(downstream.Spec.Lifecycle) {
		var err error
		downstream, err = r.copyPublished(ctx, downstream, dp)
		if err != nil {
			return nil, err
		}
	}

	return r.updateDraft(ctx, downstream, upstream)
}

func (r *DownstreamPackageReconciler) getDownstreamPR(dp *api.DownstreamPackage,
	prList *porchapi.PackageRevisionList) *porchapi.PackageRevision {
	downstream := dp.Spec.Downstream

	var latestPublished *porchapi.PackageRevision
	// the first package revision number that porch assigns is "v1",
	// so use v0 as a placeholder for comparison
	latestVersion := "v0"

	for _, pr := range prList.Items {
		// look for the downstream package in the target repo
		if pr.Spec.RepositoryName != downstream.Repo ||
			pr.Spec.PackageName != downstream.Package {
			continue
		}
		// check that the downstream package was created by this DownstreamPackage object
		owned := false

		// TODO: Implement the "AdoptionPolicy" field, which allows the user to decide if
		//  the controller should adopt existing package revisions or ignore them. For now,
		//  we just ignore them.
		for _, owner := range pr.OwnerReferences {
			if owner.UID == dp.UID {
				owned = true
			}
		}
		if !owned {
			// this downstream package doesn't belong to us
			continue
		}

		// Check if this PR is a draft. We should only have one draft created by this controller at a time,
		// so we can just return it.
		if !porchapi.LifecycleIsPublished(pr.Spec.Lifecycle) {
			return &pr
		} else {
			latestPublished, latestVersion = compare(&pr, latestPublished, latestVersion)
		}
	}

	return latestPublished
}

func compare(pr, latestPublished *porchapi.PackageRevision, latestVersion string) (*porchapi.PackageRevision, string) {
	switch cmp := semver.Compare(pr.Spec.Revision, latestVersion); {
	case cmp == 0:
		// Same revision.
	case cmp < 0:
		// current < latest; no change
	case cmp > 0:
		// current > latest; update latest
		latestVersion = pr.Spec.Revision
		latestPublished = pr.DeepCopy()
	}
	return latestPublished, latestVersion
}

// determine if the downstream PR needs to be updated
func (r *DownstreamPackageReconciler) isUpToDate(dp *api.DownstreamPackage, downstream *porchapi.PackageRevision) bool {
	upstreamLock := downstream.Status.UpstreamLock
	lastIndex := strings.LastIndex(upstreamLock.Git.Ref, "/")
	if strings.HasPrefix(upstreamLock.Git.Ref, "drafts") {
		// the current upstream is a draft, so it needs to be updated.
		return false
	}
	currentUpstreamRevision := upstreamLock.Git.Ref[lastIndex+1:]
	return currentUpstreamRevision == dp.Spec.Upstream.Revision
}

func (r *DownstreamPackageReconciler) copyPublished(ctx context.Context,
	source *porchapi.PackageRevision,
	dp *api.DownstreamPackage) (*porchapi.PackageRevision, error) {
	tr := true
	newPR := &porchapi.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PackageRevision",
			APIVersion: porchapi.SchemeGroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: source.Namespace,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         dp.APIVersion,
				Kind:               dp.Kind,
				Name:               dp.Name,
				UID:                dp.UID,
				Controller:         &tr,
				BlockOwnerDeletion: nil,
			}},
		},
		Spec: source.Spec,
	}

	newPR.Spec.Revision = ""
	newPR.Spec.WorkspaceName = newWorkspaceName(newPR.Spec.WorkspaceName)
	newPR.Spec.Lifecycle = porchapi.PackageRevisionLifecycleDraft

	if err := r.Client.Create(ctx, newPR); err != nil {
		return nil, err
	}

	return newPR, nil
}

func newWorkspaceName(oldWorkspaceName porchapi.WorkspaceName) porchapi.WorkspaceName {
	wsNumStr := strings.TrimPrefix(string(oldWorkspaceName), workspaceNamePrefix)
	wsNum, _ := strconv.Atoi(wsNumStr)
	wsNum++
	return porchapi.WorkspaceName(fmt.Sprintf(workspaceNamePrefix+"%d", wsNum))
}

func (r *DownstreamPackageReconciler) updateDraft(ctx context.Context,
	draft *porchapi.PackageRevision,
	newUpstreamPR *porchapi.PackageRevision) (*porchapi.PackageRevision, error) {

	draft = draft.DeepCopy()
	tasks := draft.Spec.Tasks

	updateTask := porchapi.Task{
		Type: porchapi.TaskTypeUpdate,
		Update: &porchapi.PackageUpdateTaskSpec{
			Upstream: tasks[0].Clone.Upstream,
		},
	}
	updateTask.Update.Upstream.UpstreamRef.Name = newUpstreamPR.Name
	draft.Spec.Tasks = append(tasks, updateTask)

	err := r.Client.Update(ctx, draft)
	if err != nil {
		return nil, err
	}
	return draft, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DownstreamPackageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := porchapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := configapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.DownstreamPackage{}).
		Complete(r)
}
