// Copyright 2022 The kpt Authors
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

package rootsyncrollout

import (
	"context"
	"flag"
	"fmt"
	"sync"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	rsdapi "github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncdeployments/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncrollouts/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	rootSyncRolloutLabel = "config.porch.kpt.dev/rollout"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

func NewRootSyncRolloutReconciler() *RootSyncRolloutReconciler {
	return &RootSyncRolloutReconciler{
		packageTargetCache: make(map[types.NamespacedName]v1alpha1.PackageSelector),
	}
}

// RootSyncRolloutReconciler reconciles a RootSyncRollout object
type RootSyncRolloutReconciler struct {
	Options

	client.Client

	mutex              sync.Mutex
	packageTargetCache map[types.NamespacedName]v1alpha1.PackageSelector
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-rootsyncrollouts webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncrollouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncrollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncrollouts/finalizers,verbs=update
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncdeployments,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RootSyncRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Running reconcile")
	var rootsyncrollout v1alpha1.RootSyncRollout
	if err := r.Get(ctx, req.NamespacedName, &rootsyncrollout); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	myFinalizerName := "config.porch.kpt.dev/rootsyncrollouts"
	if rootsyncrollout.ObjectMeta.DeletionTimestamp.IsZero() {
		// Update the cache with mapping from rollouts to PackageRevision targets. It allows the controller
		// to determine which Rollouts needs to be reconciled based on an event about a cluster.
		func() {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.packageTargetCache[req.NamespacedName] = rootsyncrollout.Spec.Packages
		}()

		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&rootsyncrollout, myFinalizerName) {
			controllerutil.AddFinalizer(&rootsyncrollout, myFinalizerName)
			if err := r.Update(ctx, &rootsyncrollout); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// Clean up the cache.
		func() {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			delete(r.packageTargetCache, req.NamespacedName)
		}()

		// The object is being deleted
		if controllerutil.ContainsFinalizer(&rootsyncrollout, myFinalizerName) {
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&rootsyncrollout, myFinalizerName)
			if err := r.Update(ctx, &rootsyncrollout); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Get all packages targeted by the Package selector
	packageMap, err := r.getPackages(ctx, &rootsyncrollout)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Looked up packages", "count", len(packageMap))

	rootSyncDeployments, err := r.listRootSyncDeployments(ctx, rootsyncrollout.Name, rootsyncrollout.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	logger.Info("Looked up RootSyncDeployments", "count", len(rootSyncDeployments))

	currentRsds := make(map[string][]*rsdapi.RootSyncDeployment)
	for i := range rootSyncDeployments {
		rsd := rootSyncDeployments[i]

		pkgName, found := usesPackage(rsd, packageMap)
		if !found {
			logger.Info("Package not found, deleting RootSyncDeployment", "rsd", rsd.Name)
			if r.Delete(ctx, rsd); err != nil {
				return ctrl.Result{}, err
			}
			continue
		}
		logger.Info("Found package", "pkg", pkgName)

		latestPkgRev, found := findLatest(packageMap[pkgName])
		if !found {
			return ctrl.Result{}, fmt.Errorf("no PackageRevision tagged with `latest` found for package %s", pkgName)
		}
		logger.Info("Found latest packagerevision", "pkgRev", latestPkgRev.Name)
		newRsd := newRootSyncDeployment(&rootsyncrollout, pkgName, latestPkgRev.Name)
		if !equality.Semantic.DeepEqual(rsd.Spec, newRsd.Spec) {
			rsd.Spec = newRsd.Spec
			if err := r.Update(ctx, rsd); err != nil {
				return ctrl.Result{}, err
			}
		}
		currentRsds[pkgName] = append(currentRsds[pkgName], rsd)
	}

	for pkgName, pkgRevs := range packageMap {
		var found bool
		for _, rsd := range rootSyncDeployments {
			rsdPkgRevNamespace := rsd.Spec.PackageRevision.Namespace
			rsdPkgRevName := rsd.Spec.PackageRevision.Name
			if rsdPkgRevNamespace == "" {
				rsdPkgRevNamespace = rsd.Namespace
			}
			for _, pkgRev := range pkgRevs {
				if pkgRev.Name == rsdPkgRevName && pkgRev.Namespace == rsdPkgRevNamespace {
					found = true
				}
			}
		}
		if !found {
			logger.Info("No RootSyncDeployment found for package", "pkg", pkgName)
			pkgRev, found := findLatest(pkgRevs)
			if !found {
				return ctrl.Result{}, fmt.Errorf("no PackageRevision tagged with `latest` found for package %s", pkgName)
			}
			rsd := newRootSyncDeployment(&rootsyncrollout, pkgName, pkgRev.Name)
			if err := r.Create(ctx, rsd); err != nil {
				return ctrl.Result{}, fmt.Errorf("error creating new RootSyncDeployment %s: %v", rsd.Name, err)
			}
			currentRsds[pkgName] = append(currentRsds[pkgName], rsd)
		}
	}

	if err := r.updateStatus(ctx, &rootsyncrollout, currentRsds); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RootSyncRolloutReconciler) updateStatus(ctx context.Context, rsr *v1alpha1.RootSyncRollout, rsdsMap map[string][]*rsdapi.RootSyncDeployment) error {
	var packageStatuses []v1alpha1.PackageStatus

	for pkgName, rsds := range rsdsMap {
		pkgStatus := v1alpha1.PackageStatus{
			Package: pkgName,
		}

		revisionMap := make(map[string][]rsdapi.ClusterRefStatus)
		for i := range rsds {
			rsd := rsds[i]
			for j := range rsd.Status.ClusterRefStatuses {
				crs := rsd.Status.ClusterRefStatuses[j]
				revision := crs.Revision
				revisionMap[revision] = append(revisionMap[revision], crs)
			}
		}

		for rev, crss := range revisionMap {
			revision := v1alpha1.Revision{
				Revision: rev,
			}

			var count int
			var syncedCount int
			for _, crs := range crss {
				count += 1
				if crs.Synced {
					syncedCount += 1
				}
			}
			revision.Count = count
			revision.SyncedCount = syncedCount
			pkgStatus.Revisions = append(pkgStatus.Revisions, revision)
		}

		packageStatuses = append(packageStatuses, pkgStatus)
	}

	if equality.Semantic.DeepEqual(rsr.Status.PackageStatuses, packageStatuses) {
		klog.Infof("Status has not changed, update not needed.")
		return nil
	}
	rsr.Status.PackageStatuses = packageStatuses
	return r.Status().Update(ctx, rsr)
}

func usesPackage(rsd *rsdapi.RootSyncDeployment, packageMap map[string][]*porchapi.PackageRevision) (string, bool) {
	for pkgName, pkgRevs := range packageMap {
		for _, pkgRev := range pkgRevs {
			if pkgRev.Name == rsd.Spec.PackageRevision.Name {
				return pkgName, true
			}
		}
	}
	return "", false
}

func findLatest(pkgRevs []*porchapi.PackageRevision) (*porchapi.PackageRevision, bool) {
	for _, pr := range pkgRevs {
		if pr.Labels[porchapi.LatestPackageRevisionKey] == porchapi.LatestPackageRevisionValue {
			return pr, true
		}
	}
	return nil, false
}

func newRootSyncDeployment(rsr *v1alpha1.RootSyncRollout, pkgName, pkgRevision string) *rsdapi.RootSyncDeployment {
	t := true
	return &rsdapi.RootSyncDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgName,
			Namespace: rsr.Namespace,
			Labels: map[string]string{
				rootSyncRolloutLabel: rsr.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: rsr.APIVersion,
					Kind:       rsr.Kind,
					Name:       rsr.Name,
					UID:        rsr.UID,
					Controller: &t,
				},
			},
		},
		Spec: rsdapi.RootSyncDeploymentSpec{
			Targets: rsdapi.ClusterTargetSelector{
				Selector: rsr.Spec.Targets.Selector,
			},
			PackageRevision: rsdapi.PackageRevisionRef{
				Name:      pkgRevision,
				Namespace: rsr.Namespace,
			},
		},
	}
}

func (r *RootSyncRolloutReconciler) getPackages(ctx context.Context, rollout *v1alpha1.RootSyncRollout) (map[string][]*porchapi.PackageRevision, error) {
	var packageRevisionList porchapi.PackageRevisionList
	var opts []client.ListOption
	if rollout.Spec.Packages.Namespace != "" {
		opts = append(opts, client.InNamespace(rollout.Spec.Packages.Namespace))
	}
	if rollout.Spec.Packages.Selector != nil {
		selector, err := metav1.LabelSelectorAsSelector(rollout.Spec.Packages.Selector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}
	if err := r.List(ctx, &packageRevisionList, opts...); err != nil {
		return nil, err
	}

	packageMap := make(map[string][]*porchapi.PackageRevision)
	for i := range packageRevisionList.Items {
		packageRevision := packageRevisionList.Items[i]
		packageName := packageRevision.Spec.PackageName
		// TODO: See if we can handle this with a FieldSelector on packagerevisions.
		if !porchapi.LifecycleIsPublished(packageRevision.Spec.Lifecycle) {
			continue
		}
		if _, found := packageMap[packageName]; !found {
			packageMap[packageName] = make([]*porchapi.PackageRevision, 0)
		}
		packageMap[packageName] = append(packageMap[packageName], &packageRevision)
	}

	return packageMap, nil
}

func (r *RootSyncRolloutReconciler) listRootSyncDeployments(ctx context.Context, rsrName, rsrNamespace string) ([]*rsdapi.RootSyncDeployment, error) {
	var list rsdapi.RootSyncDeploymentList
	if err := r.List(ctx, &list, client.MatchingLabels{rootSyncRolloutLabel: rsrName}, client.InNamespace(rsrNamespace)); err != nil {
		return nil, err
	}
	var rsds []*rsdapi.RootSyncDeployment
	for i := range list.Items {
		item := &list.Items[i]
		rsds = append(rsds, item)
	}
	return rsds, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootSyncRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	var pkgRev porchapi.PackageRevision

	r.Client = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RootSyncRollout{}).
		Owns(&rsdapi.RootSyncDeployment{}).
		Watches(
			&source.Kind{Type: &pkgRev},
			handler.EnqueueRequestsFromMapFunc(r.findRolloutsForPackageRevision),
		).
		Complete(r)
}

func (r *RootSyncRolloutReconciler) findRolloutsForPackageRevision(pkgRev client.Object) []reconcile.Request {
	// There is not support for reverse lookup by label: https://github.com/kubernetes/kubernetes/issues/1348
	var requests []reconcile.Request
	l := pkgRev.GetLabels()
	for nn, packageTargetSelector := range r.packageTargetCache {
		selector, _ := metav1.LabelSelectorAsSelector(packageTargetSelector.Selector)
		if !(packageTargetSelector.Selector == nil || selector.Empty() || selector.Matches(labels.Set(l))) {
			continue
		}
		if !(packageTargetSelector.Namespace == "" || packageTargetSelector.Namespace == pkgRev.GetNamespace()) {
			continue
		}

		requests = append(requests, reconcile.Request{NamespacedName: nn})
	}
	return requests
}
