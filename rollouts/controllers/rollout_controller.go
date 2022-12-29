/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"flag"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/clusterstore"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/packageclustermatcher"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/packagediscovery"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

const (
	rolloutLabel = "gitops.kpt.dev/rollout-name"
)

// RolloutReconciler reconciles a Rollout object
type RolloutReconciler struct {
	client.Client

	store *clusterstore.ClusterStore

	Scheme *runtime.Scheme

	mutex                 sync.Mutex
	packageDiscoveryCache map[types.NamespacedName]*packagediscovery.PackageDiscovery
}

//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Rollout object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
//
// Dumb reconciliation of Rollout API includes the following:
// Fetch the READY kcc clusters.
// For each kcc cluster, fetch RootSync objects in each of the KCC clusters.
func (r *RolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var rollout gitopsv1alpha1.Rollout

	logger.Info("reconciling", "key", req.NamespacedName)

	if err := r.Get(ctx, req.NamespacedName, &rollout); err != nil {
		logger.Error(err, "unable to fetch Rollout")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	myFinalizerName := "gitops.kpt.dev/rollouts"
	if rollout.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&rollout, myFinalizerName) {
			controllerutil.AddFinalizer(&rollout, myFinalizerName)
			if err := r.Update(ctx, &rollout); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&rollout, myFinalizerName) {
			func() {
				r.mutex.Lock()
				defer r.mutex.Unlock()

				// clean cache
				delete(r.packageDiscoveryCache, req.NamespacedName)
			}()

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&rollout, myFinalizerName)
			if err := r.Update(ctx, &rollout); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	packageDiscoveryClient := func() *packagediscovery.PackageDiscovery {
		r.mutex.Lock()
		defer r.mutex.Unlock()

		client, found := r.packageDiscoveryCache[req.NamespacedName]
		if !found {
			client = packagediscovery.NewPackageDiscovery(r.Client, rollout.Namespace)
			r.packageDiscoveryCache[req.NamespacedName] = client
		}

		return client
	}()

	err := r.reconcileRollout(ctx, &rollout, packageDiscoveryClient)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RolloutReconciler) reconcileRollout(ctx context.Context, rollout *gitopsv1alpha1.Rollout, packageDiscoveryClient *packagediscovery.PackageDiscovery) error {
	logger := log.FromContext(ctx)

	clusters, err := r.store.ListClusters(ctx, rollout.Spec.Targets.Selector)
	if err != nil {
		return err
	}
	logger.Info("discovered clusters", "count", len(clusters.Items))

	discoveredPackages, err := packageDiscoveryClient.GetPackages(ctx, rollout.Spec.Packages)
	if err != nil {
		logger.Error(err, "failed to discover packages")
		return client.IgnoreNotFound(err)
	}
	logger.Info("discovered packages", "count", len(discoveredPackages), "packages", discoveredPackages)

	packageClusterMatcherClient := packageclustermatcher.NewPackageClusterMatcher(clusters.Items, discoveredPackages)

	allClusterPackages, err := packageClusterMatcherClient.GetClusterPackages(rollout.Spec.PackageToTargetMatcher)
	if err != nil {
		logger.Error(err, "get cluster packages failed")
		return client.IgnoreNotFound(err)
	}

	for _, clusterPackages := range allClusterPackages {
		clusterName := clusterPackages.Cluster.Name
		logger.Info("cluster packages", "cluster", clusterName, "packagesCount", len(clusterPackages.Packages), "packages", clusterPackages.Packages)
	}

	targets, err := r.computeTargets(ctx, rollout, allClusterPackages)
	if err != nil {
		return err
	}

	clusterStatuses, err := r.rolloutTargets(ctx, rollout, targets)
	if err != nil {
		return err
	}

	if err := r.updateStatus(ctx, rollout, clusterStatuses); err != nil {
		return err
	}
	return nil
}

func (r *RolloutReconciler) updateStatus(ctx context.Context, rollout *gitopsv1alpha1.Rollout, clusterStatuses []gitopsv1alpha1.ClusterStatus) error {
	logger := log.FromContext(ctx)
	logger.Info("updating the status", "cluster statuses", len(clusterStatuses))
	rollout.Status.ClusterStatuses = clusterStatuses
	rollout.Status.ObservedGeneration = rollout.Generation
	return r.Client.Status().Update(ctx, rollout)
}

/*
so we compute targets where each target consists of (cluster, package)
compute the RRS corresponding to each (cluster, package) pair
For RRS, name has to be function of cluster-id and package-id.
For RRS, make rootSyncTemplate
*/
func (r *RolloutReconciler) computeTargets(ctx context.Context,
	rollout *gitopsv1alpha1.Rollout,
	clusterPackages []packageclustermatcher.ClusterPackages) (*Targets, error) {

	RRSkeysToBeDeleted := map[client.ObjectKey]*gitopsv1alpha1.RemoteRootSync{}
	// let's take a look at existing remoterootsyncs
	existingRRSs, err := r.listRemoteRootSyncs(ctx, rollout.Name, rollout.Namespace)
	if err != nil {
		return nil, err
	}
	// initially assume all the keys to be deleted
	for _, rrs := range existingRRSs {
		RRSkeysToBeDeleted[client.ObjectKeyFromObject(rrs)] = rrs
	}
	klog.Infof("Found remoterootsyncs: %s", toRemoteRootSyncNames(existingRRSs))
	targets := &Targets{}
	// track keys of all the desired remote rootsyncs
	for _, clusterPkg := range clusterPackages {
		// TODO: figure out multiple packages per cluster story
		if len(clusterPkg.Packages) < 1 {
			continue
		}
		cluster := &clusterPkg.Cluster
		pkg := &clusterPkg.Packages[0]
		rrs := gitopsv1alpha1.RemoteRootSync{}
		key := client.ObjectKey{
			Namespace: rollout.Namespace,
			Name:      fmt.Sprintf("%s-%s", pkgID(pkg), cluster.Name),
		}
		// since this RRS need to exist, remove it from the deletion list
		delete(RRSkeysToBeDeleted, key)
		// check if this remoterootsync for this package exists or not ?
		err := r.Client.Get(ctx, key, &rrs)
		if err != nil {
			if apierrors.IsNotFound(err) { // rrs is missing
				targets.ToBeCreated = append(targets.ToBeCreated, &clusterPackagePair{
					cluster:    cluster,
					packageRef: pkg,
				})
			} else {
				// some other error encountered
				return nil, err
			}
		} else {
			// remoterootsync already exists
			if pkg.Revision != rrs.Spec.Template.Spec.Git.Revision {
				rrs.Spec.Template.Spec.Git.Revision = pkg.Revision
				// revision has been updated
				targets.ToBeUpdated = append(targets.ToBeUpdated, &rrs)
			} else {
				targets.Unchanged = append(targets.Unchanged, &rrs)
			}
		}
	}

	for _, rrs := range RRSkeysToBeDeleted {
		targets.ToBeDeleted = append(targets.ToBeDeleted, rrs)
	}

	return targets, nil
}

func (r *RolloutReconciler) rolloutTargets(ctx context.Context, rollout *gitopsv1alpha1.Rollout, targets *Targets) ([]gitopsv1alpha1.ClusterStatus, error) {

	clusterStatuses := []gitopsv1alpha1.ClusterStatus{}

	if rollout.Spec.Strategy.Type != gitopsv1alpha1.AllAtOnce {
		return clusterStatuses, fmt.Errorf("%v strategy not supported yet", rollout.Spec.Strategy.Type)
	}

	for _, target := range targets.ToBeCreated {
		rootSyncSpec := toRootSyncSpec(target.packageRef)
		rrs := newRemoteRootSync(rollout,
			gitopsv1alpha1.ClusterRef{Name: target.cluster.Name},
			rootSyncSpec,
			pkgID(target.packageRef),
		)
		if err := r.Create(ctx, rrs); err != nil {
			klog.Warningf("Error creating RemoteRootSync %s: %v", rrs.Name, err)
			return nil, err
		}
		clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
			Name: rrs.Spec.ClusterRef.Name,
			PackageStatus: gitopsv1alpha1.PackageStatus{
				PackageID:  rrs.Name,
				SyncStatus: rrs.Status.SyncStatus,
			},
		})
	}

	for _, target := range targets.ToBeUpdated {
		if err := r.Update(ctx, target); err != nil {
			klog.Warningf("Error updating RemoteRootSync %s: %v", target.Name, err)
			return nil, err
		}
		clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
			Name: target.Spec.ClusterRef.Name,
			PackageStatus: gitopsv1alpha1.PackageStatus{
				PackageID:  target.Name,
				SyncStatus: target.Status.SyncStatus,
			},
		})
	}

	for _, target := range targets.ToBeDeleted {
		if err := r.Delete(ctx, target); err != nil {
			klog.Warningf("Error deleting RemoteRootSync %s: %v", target.Name, err)
			return nil, err
		}
	}

	for _, target := range targets.Unchanged {
		clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
			Name: target.Spec.ClusterRef.Name,
			PackageStatus: gitopsv1alpha1.PackageStatus{
				PackageID:  target.Name,
				SyncStatus: target.Status.SyncStatus,
			},
		})
	}

	return clusterStatuses, nil
}

type Targets struct {
	ToBeCreated []*clusterPackagePair
	ToBeUpdated []*gitopsv1alpha1.RemoteRootSync
	ToBeDeleted []*gitopsv1alpha1.RemoteRootSync
	Unchanged   []*gitopsv1alpha1.RemoteRootSync
}

type clusterPackagePair struct {
	cluster    *gkeclusterapis.ContainerCluster
	packageRef *packagediscovery.DiscoveredPackage
}

func toRemoteRootSyncNames(rsss []*gitopsv1alpha1.RemoteRootSync) []string {
	var names []string
	for _, rss := range rsss {
		names = append(names, rss.Name)
	}
	return names
}

func (r *RolloutReconciler) testClusterClient(ctx context.Context, cl client.Client) error {
	logger := log.FromContext(ctx)
	podList := &v1.PodList{}
	err := cl.List(context.Background(), podList, client.InNamespace("kube-system"))
	if err != nil {
		return err
	}
	logger.Info("found podlist", "number of pods", len(podList.Items))
	return nil
}

func (r *RolloutReconciler) listRemoteRootSyncs(ctx context.Context, rsdName, rsdNamespace string) ([]*gitopsv1alpha1.RemoteRootSync, error) {
	var list gitopsv1alpha1.RemoteRootSyncList
	if err := r.List(ctx, &list, client.MatchingLabels{rolloutLabel: rsdName}, client.InNamespace(rsdNamespace)); err != nil {
		return nil, err
	}
	var remoterootsyncs []*gitopsv1alpha1.RemoteRootSync
	for i := range list.Items {
		item := &list.Items[i]
		remoterootsyncs = append(remoterootsyncs, item)
	}
	return remoterootsyncs, nil
}

func isRRSSynced(rss *gitopsv1alpha1.RemoteRootSync) bool {
	if rss.Generation != rss.Status.ObservedGeneration {
		return false
	}

	if rss.Status.SyncStatus == "Synced" {
		return true
	}
	return false
}

// Given a package identifier and cluster, create a RemoteRootSync object.
func newRemoteRootSync(rollout *gitopsv1alpha1.Rollout, clusterRef gitopsv1alpha1.ClusterRef, rssSpec *gitopsv1alpha1.RootSyncSpec, pkgID string) *gitopsv1alpha1.RemoteRootSync {
	t := true
	return &gitopsv1alpha1.RemoteRootSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", pkgID, clusterRef.Name),
			Namespace: rollout.Namespace,
			Labels: map[string]string{
				rolloutLabel: rollout.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: rollout.APIVersion,
					Kind:       rollout.Kind,
					Name:       rollout.Name,
					UID:        rollout.UID,
					Controller: &t,
				},
			},
		},
		Spec: gitopsv1alpha1.RemoteRootSyncSpec{
			ClusterRef: clusterRef,
			Template: &gitopsv1alpha1.RootSyncInfo{
				Spec: rssSpec,
			},
		},
	}
}

func toRootSyncSpec(dpkg *packagediscovery.DiscoveredPackage) *gitopsv1alpha1.RootSyncSpec {
	return &gitopsv1alpha1.RootSyncSpec{
		SourceFormat: "unstructured",
		Git: &gitopsv1alpha1.GitInfo{
			Repo:     fmt.Sprintf("https://github.com/%s/%s.git", dpkg.Org, dpkg.Repo),
			Revision: dpkg.Revision,
			Dir:      dpkg.Directory,
			Branch:   "main",
			Auth:     "none",
		},
	}
}

func pkgID(dpkg *packagediscovery.DiscoveredPackage) string {
	return fmt.Sprintf("%s-%s-%s", dpkg.Org, dpkg.Repo, dpkg.Directory)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := gkeclusterapis.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := gitopsv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	r.Client = mgr.GetClient()

	r.packageDiscoveryCache = make(map[types.NamespacedName]*packagediscovery.PackageDiscovery)

	// setup the clusterstore
	r.store = &clusterstore.ClusterStore{
		Config: mgr.GetConfig(),
		Client: r.Client,
	}
	if err := r.store.Init(); err != nil {
		return err
	}
	// TODO: watch cluster resources as well
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1alpha1.Rollout{}).
		Owns(&gitopsv1alpha1.RemoteRootSync{}).
		Complete(r)
}
