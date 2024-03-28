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

package rootsyncdeployment

import (
	"context"
	"flag"
	"fmt"
	"sync"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncdeployments/api/v1alpha1"
	rssapi "github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncsets/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	rootSyncSetLabel = "config.porch.kpt.dev/rootsyncdeployment"
)

var (
	configConnectorContainerClusterGVK = schema.GroupVersionKind{
		Group:   "container.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "ContainerCluster",
	}
	configConnectorConfigControllerClusterGVK = schema.GroupVersionKind{
		Group:   "configcontroller.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "ConfigControllerInstance",
	}
	rootSyncSetGVK = schema.GroupVersionKind{
		Group:   "config.porch.kpt.dev",
		Version: "v1alpha1",
		Kind:    "RootSyncSet",
	}
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

func NewRootSyncDeploymentReconciler() *RootSyncDeploymentReconciler {
	return &RootSyncDeploymentReconciler{
		clusterTargetCache: make(map[types.NamespacedName]labels.Selector),
	}
}

// RootSyncDeploymentReconciler reconciles a RootSyncDeployment object
type RootSyncDeploymentReconciler struct {
	Options

	client.Client

	mutex              sync.Mutex
	clusterTargetCache map[types.NamespacedName]labels.Selector
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-rootsyncdeployments webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=repositories,verbs=get;list;watch

//+kubebuilder:rbac:groups=configcontroller.cnrm.cloud.google.com,resources=configcontrollerinstances,verbs=get;list;watch
//+kubebuilder:rbac:groups=container.cnrm.cloud.google.com,resources=containerclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=gkehub.cnrm.cloud.google.com,resources=gkehubmemberships,verbs=get;list;watch

//+kubebuilder:rbac:groups=core.cnrm.cloud.google.com,resources=configconnectors;configconnectorcontexts,verbs=get;list;watch

func (r *RootSyncDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var rootsyncdeployment v1alpha1.RootSyncDeployment
	if err := r.Get(ctx, req.NamespacedName, &rootsyncdeployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	myFinalizerName := "config.porch.kpt.dev/rootsyncdeployments"
	if rootsyncdeployment.ObjectMeta.DeletionTimestamp.IsZero() {
		// Update the cache with mapping from rollouts to ClusterTargets. It allows the controller
		// to determine which Rollouts needs to be reconciled based on an event about a cluster.
		selector, err := metav1.LabelSelectorAsSelector(rootsyncdeployment.Spec.Targets.Selector)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("error converting targets selector to labels.Selector: %v", err)
		}
		func() {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			r.clusterTargetCache[req.NamespacedName] = selector
		}()

		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&rootsyncdeployment, myFinalizerName) {
			controllerutil.AddFinalizer(&rootsyncdeployment, myFinalizerName)
			if err := r.Update(ctx, &rootsyncdeployment); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// Clean up the cache.
		func() {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			delete(r.clusterTargetCache, req.NamespacedName)
		}()

		// The object is being deleted
		if controllerutil.ContainsFinalizer(&rootsyncdeployment, myFinalizerName) {
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&rootsyncdeployment, myFinalizerName)
			if err := r.Update(ctx, &rootsyncdeployment); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	res, err := r.syncRootSyncDeployment(ctx, &rootsyncdeployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, &rootsyncdeployment, res); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RootSyncDeploymentReconciler) updateStatus(ctx context.Context, rsd *v1alpha1.RootSyncDeployment, clusterRefStatuses []v1alpha1.ClusterRefStatus) error {
	if equality.Semantic.DeepEqual(rsd.Status.ClusterRefStatuses, clusterRefStatuses) {
		klog.Infof("Status has not changed, update not needed.")
		return nil
	}
	rsd.Status.ObservedGeneration = rsd.Generation
	rsd.Status.ClusterRefStatuses = clusterRefStatuses
	return r.Status().Update(ctx, rsd)
}

func (r *RootSyncDeploymentReconciler) syncRootSyncDeployment(ctx context.Context, rsd *v1alpha1.RootSyncDeployment) ([]v1alpha1.ClusterRefStatus, error) {
	var clusterRefStatuses []v1alpha1.ClusterRefStatus
	namespace := rsd.Spec.PackageRevision.Namespace
	if namespace == "" {
		namespace = rsd.Namespace
	}
	nn := types.NamespacedName{
		Name:      rsd.Spec.PackageRevision.Name,
		Namespace: namespace,
	}

	pkgRev, err := r.getPackageRevision(ctx, nn)
	if err != nil {
		return clusterRefStatuses, err
	}

	repo, err := r.getRepository(ctx, types.NamespacedName{
		Name:      pkgRev.Spec.RepositoryName,
		Namespace: pkgRev.Namespace,
	})
	if err != nil {
		return clusterRefStatuses, err
	}

	rootSyncSetSpec := toRootSyncSpec(pkgRev, repo)

	clusters, err := r.getClusters(ctx, rsd.Spec.Targets.Selector)
	if err != nil {
		return clusterRefStatuses, err
	}
	klog.Infof("Found clusters: %s", toClusterNames(clusters))

	rootsyncsets, err := r.listRootSyncSets(ctx, rsd.Name, rsd.Namespace)
	if err != nil {
		return clusterRefStatuses, err
	}
	klog.Infof("Found RootSyncSets: %s", toRootSyncSetNames(rootsyncsets))

	var clustersMissingRootSync []rssapi.ClusterRef
	var clustersToDeleteRootSync []*rssapi.RootSyncSet
	var clustersNeedingUpdate []*rssapi.RootSyncSet
	var clustersBeingUpdated []*rssapi.RootSyncSet
	var upToDateClusters []*rssapi.RootSyncSet

	for i := range rootsyncsets {
		rss := rootsyncsets[i]

		// TODO: See if we can remove this constraint. But it make things easier for now.
		// First remove RootSyncSets for clusters where it should no longer exist.
		if l := len(rss.Spec.ClusterRefs); l != 1 {
			return clusterRefStatuses,
				fmt.Errorf("RootSyncSet %s contains %d ClusterRefs, but it should be 1", rss.Name, l)
		}

		// If the RootSyncSet references a cluster that is no longer matches by the target selector,
		// we need to delete the RootSyncSet.
		clusterRef := rss.Spec.ClusterRefs[0]
		if !contains(clusters, clusterRef) {
			clustersToDeleteRootSync = append(clustersToDeleteRootSync, rss)
			continue
		}

		// If the RootSync template in the RootSyncSet doesn't match the spec from the
		// PackageRevision, it needs to be updated.
		if !equality.Semantic.DeepEqual(rootSyncSetSpec, rss.Spec.Template.Spec) {
			clustersNeedingUpdate = append(clustersNeedingUpdate, rss)
			continue
		}

		// If the RootSyncSet is not synced, we treat them as being the process of being updated.
		// This might not be correct in all situations.
		if !isRssSynced(rss) {
			clustersBeingUpdated = append(clustersBeingUpdated, rss)
			continue
		}
		upToDateClusters = append(upToDateClusters, rss)
	}

	// Find any clusters that are targeted by the label selector, but
	// where we don't have a RootSyncSet referencing the cluster.
	for i := range clusters {
		cluster := clusters[i]
		ref := rssapi.ClusterRef{
			Name:       cluster.GetName(),
			Namespace:  cluster.GetNamespace(),
			ApiVersion: cluster.GetAPIVersion(),
			Kind:       cluster.GetKind(),
		}
		var found bool
		for _, rss := range rootsyncsets {
			rssClusterRef := *rss.Spec.ClusterRefs[0]
			if equality.Semantic.DeepEqual(rssClusterRef, ref) {
				found = true
			}
		}
		if !found {
			clustersMissingRootSync = append(clustersMissingRootSync, ref)
		}
	}

	// TODO: We should keep track of the clusters and status until we know they have been removed
	// from the cluster.
	for _, rss := range clustersToDeleteRootSync {
		if err := r.Delete(ctx, rss); err != nil {
			klog.Warningf("Error removing RootSyncSet %s: %v", rss.Name, err)
		}
	}

	// If we have clusters that doesn't already have the package installed, we just
	// update them all. The limit to the number of clusters being updated
	// concurrently (currently that is 1) does not apply here.
	if len(clustersMissingRootSync) > 0 {
		for _, cr := range clustersMissingRootSync {
			rss := newRootSyncSet(rsd, cr, rootSyncSetSpec, pkgRev.Spec.PackageName)
			if err := r.Create(ctx, rss); err != nil {
				klog.Warningf("Error creating RootSyncSet %s: %v", rss.Name, err)
			}
			clustersBeingUpdated = append(clustersBeingUpdated, rss)
		}
	}

	// If no clusters are in the process of being updated and we have clusters
	// that needs to be updated, just update the corresponding RootSyncSet.
	if len(clustersBeingUpdated) == 0 && len(clustersNeedingUpdate) > 0 {
		rss := clustersNeedingUpdate[0]
		rss.Spec.Template.Spec = rootSyncSetSpec
		if err := r.Update(ctx, rss); err != nil {
			klog.Warningf("Error updating RootSyncSet %s: %v", rss.Name, err)
		}
		clustersNeedingUpdate = clustersNeedingUpdate[1:]
		clustersBeingUpdated = append(clustersBeingUpdated, rss)
	}

	for _, c := range upToDateClusters {
		clusterRefStatuses = append(clusterRefStatuses, newClusterRefStatus(c, true))
	}
	for _, c := range clustersBeingUpdated {
		clusterRefStatuses = append(clusterRefStatuses, newClusterRefStatus(c, false))
	}
	for _, c := range clustersNeedingUpdate {
		clusterRefStatuses = append(clusterRefStatuses, newClusterRefStatus(c, isRssSynced(c)))
	}

	return clusterRefStatuses, nil
}

func newClusterRefStatus(rss *rssapi.RootSyncSet, synced bool) v1alpha1.ClusterRefStatus {
	clusterRef := rss.Spec.ClusterRefs[0]
	return v1alpha1.ClusterRefStatus{
		ApiVersion: clusterRef.ApiVersion,
		Kind:       clusterRef.Kind,
		Name:       clusterRef.Name,
		Namespace:  clusterRef.Namespace,
		Revision:   rss.Spec.Template.Spec.Git.Revision,
		Synced:     synced,
	}
}

func (r *RootSyncDeploymentReconciler) getPackageRevision(ctx context.Context, nn types.NamespacedName) (*porchapi.PackageRevision, error) {
	var packageRevision porchapi.PackageRevision
	if err := r.Get(ctx, nn, &packageRevision); err != nil {
		return nil, err
	}
	return &packageRevision, nil
}

func (r *RootSyncDeploymentReconciler) getRepository(ctx context.Context, nn types.NamespacedName) (*configapi.Repository, error) {
	var repository configapi.Repository
	if err := r.Get(ctx, nn, &repository); err != nil {
		return nil, err
	}
	return &repository, nil
}

func newRootSyncSet(rsd *v1alpha1.RootSyncDeployment, clusterRef rssapi.ClusterRef, rssSpec *rssapi.RootSyncSpec, pkgName string) *rssapi.RootSyncSet {
	t := true
	rootsyncset := &rssapi.RootSyncSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", pkgName, clusterRef.Name),
			Namespace: rsd.Namespace,
			Labels: map[string]string{
				rootSyncSetLabel: rsd.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: rsd.APIVersion,
					Kind:       rsd.Kind,
					Name:       rsd.Name,
					UID:        rsd.UID,
					Controller: &t,
				},
			},
		},
		Spec: rssapi.RootSyncSetSpec{
			ClusterRefs: []*rssapi.ClusterRef{
				&clusterRef,
			},
			Template: &rssapi.RootSyncInfo{
				Spec: rssSpec,
			},
		},
	}
	return rootsyncset
}

func toRootSyncSpec(pkgRev *porchapi.PackageRevision, repo *configapi.Repository) *rssapi.RootSyncSpec {
	return &rssapi.RootSyncSpec{
		SourceFormat: "unstructured",
		Git: &rssapi.GitInfo{
			Repo:     repo.Spec.Git.Repo,
			Revision: fmt.Sprintf("%s/%s", pkgRev.Spec.PackageName, pkgRev.Spec.Revision),
			Dir:      pkgRev.Spec.PackageName,
			Branch:   repo.Spec.Git.Branch,
			Auth:     "none",
		},
	}
}

func isRssSynced(rss *rssapi.RootSyncSet) bool {
	if rss.Generation != rss.Status.ObservedGeneration {
		return false
	}

	for _, crs := range rss.Status.ClusterRefStatuses {
		if crs.SyncStatus == "Synced" {
			return true
		}
	}
	return false
}

func contains(clusters []*unstructured.Unstructured, clusterRef *rssapi.ClusterRef) bool {
	for _, cr := range clusters {
		if clusterRef.ApiVersion == cr.GetAPIVersion() &&
			clusterRef.Kind == cr.GetKind() &&
			clusterRef.Name == cr.GetName() &&
			clusterRef.Namespace == cr.GetNamespace() {
			return true
		}
	}
	return false
}

func (r *RootSyncDeploymentReconciler) listRootSyncSets(ctx context.Context, rsdName, rsdNamespace string) ([]*rssapi.RootSyncSet, error) {
	var list rssapi.RootSyncSetList
	if err := r.List(ctx, &list, client.MatchingLabels{rootSyncSetLabel: rsdName}, client.InNamespace(rsdNamespace)); err != nil {
		return nil, err
	}
	var rootsyncsets []*rssapi.RootSyncSet
	for i := range list.Items {
		item := &list.Items[i]
		rootsyncsets = append(rootsyncsets, item)
	}
	return rootsyncsets, nil
}

func (r *RootSyncDeploymentReconciler) getClusters(ctx context.Context, sel *metav1.LabelSelector) ([]*unstructured.Unstructured, error) {
	var clusters []*unstructured.Unstructured
	for _, gvk := range []schema.GroupVersionKind{configConnectorConfigControllerClusterGVK, configConnectorContainerClusterGVK} {
		c, err := r.getClustersByGVK(ctx, gvk, sel)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c...)
	}
	return clusters, nil
}

func (r *RootSyncDeploymentReconciler) getClustersByGVK(ctx context.Context, gvk schema.GroupVersionKind, sel *metav1.LabelSelector) ([]*unstructured.Unstructured, error) {
	selector, err := metav1.LabelSelectorAsSelector(sel)
	if err != nil {
		return nil, err
	}
	var list unstructured.UnstructuredList
	list.SetGroupVersionKind(gvk)
	if err := r.List(ctx, &list, client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, err
	}

	var clusters []*unstructured.Unstructured
	for i := range list.Items {
		c := list.Items[i]
		clusters = append(clusters, &c)
	}
	return clusters, nil
}

func toRootSyncSetNames(rsss []*rssapi.RootSyncSet) []string {
	var names []string
	for _, rss := range rsss {
		names = append(names, rss.Name)
	}
	return names
}

func toClusterNames(clusters []*unstructured.Unstructured) []string {
	var names []string
	for _, c := range clusters {
		names = append(names, c.GetName())
	}
	return names
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootSyncDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := porchapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := configapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()

	var containerCluster unstructured.Unstructured
	containerCluster.SetGroupVersionKind(configConnectorContainerClusterGVK)
	var configController unstructured.Unstructured
	configController.SetGroupVersionKind(configConnectorConfigControllerClusterGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RootSyncDeployment{}).
		Owns(&rssapi.RootSyncSet{}).
		Watches(
			&source.Kind{Type: &containerCluster},
			handler.EnqueueRequestsFromMapFunc(r.requestsFromMapFunc),
		).
		Watches(
			&source.Kind{Type: &configController},
			handler.EnqueueRequestsFromMapFunc(r.requestsFromMapFunc),
		).
		Complete(r)
}

func (r *RootSyncDeploymentReconciler) requestsFromMapFunc(cluster client.Object) []reconcile.Request {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var requests []reconcile.Request
	l := cluster.GetLabels()
	for nn, clusterTargetSelector := range r.clusterTargetCache {
		if clusterTargetSelector.Matches(labels.Set(l)) {
			requests = append(requests, reconcile.Request{NamespacedName: nn})
		}
	}
	return requests
}
