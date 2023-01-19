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
	"math"
	"sort"
	"strings"
	"sync"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

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

var (
	kccClusterGVK = schema.GroupVersionKind{
		Group:   "container.cnrm.cloud.google.com",
		Version: "v1beta1",
		Kind:    "ContainerCluster",
	}
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

	strategy, err := r.getStrategy(ctx, &rollout)
	if err != nil {
		return ctrl.Result{}, err
	}

	packageDiscoveryClient := r.getPackageDiscoveryClient(req.NamespacedName)

	err = r.reconcileRollout(ctx, &rollout, strategy, packageDiscoveryClient)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RolloutReconciler) getStrategy(ctx context.Context, rollout *gitopsv1alpha1.Rollout) (*gitopsv1alpha1.ProgressiveRolloutStrategy, error) {
	logger := log.FromContext(ctx)

	progressiveStrategy := gitopsv1alpha1.ProgressiveRolloutStrategy{}
	progressiveStrategy.Spec = gitopsv1alpha1.ProgressiveRolloutStrategySpec{Waves: []gitopsv1alpha1.Wave{}}

	// validate the strategy as early as possible
	switch typ := rollout.Spec.Strategy.Type; typ {
	case gitopsv1alpha1.AllAtOnce:
		wave := gitopsv1alpha1.Wave{MaxConcurrent: math.MaxInt, Targets: rollout.Spec.Targets}
		progressiveStrategy.Spec.Waves = append(progressiveStrategy.Spec.Waves, wave)

	case gitopsv1alpha1.RollingUpdate:
		wave := gitopsv1alpha1.Wave{MaxConcurrent: rollout.Spec.Strategy.RollingUpdate.MaxConcurrent, Targets: rollout.Spec.Targets}
		progressiveStrategy.Spec.Waves = append(progressiveStrategy.Spec.Waves, wave)

	case gitopsv1alpha1.Progressive:
		strategyRef := types.NamespacedName{
			Namespace: rollout.Spec.Strategy.Progressive.Namespace,
			Name:      rollout.Spec.Strategy.Progressive.Name,
		}
		if err := r.Get(ctx, strategyRef, &progressiveStrategy); err != nil {
			logger.Error(err, "unable to fetch progressive rollout strategy", "strategyref", strategyRef)
			// TODO (droot): signal this as a condition in the rollout status
			return nil, err
		}

		err := r.validateProgressiveRolloutStrategy(ctx, rollout, &progressiveStrategy)
		if err != nil {
			logger.Error(err, "progressive rollout strategy failed validation", "strategyref", strategyRef)
			// TODO (cfry): signal this as a condition in the rollout status
			return nil, err
		}

	default:
		// TODO (droot): signal this as a condition in the rollout status
		return nil, fmt.Errorf("%v strategy not supported yet", typ)
	}

	return &progressiveStrategy, nil
}

func (r *RolloutReconciler) validateProgressiveRolloutStrategy(ctx context.Context, rollout *gitopsv1alpha1.Rollout, strategy *gitopsv1alpha1.ProgressiveRolloutStrategy) error {
	allClusters, err := r.store.ListClusters(ctx, rollout.Spec.Targets.Selector)
	if err != nil {
		return err
	}

	clusterWaveMap := make(map[string]string)
	for _, cluster := range allClusters.Items {
		clusterWaveMap[cluster.Name] = ""
	}

	pauseAfterWaveName := ""
	pauseWaveNameFound := false

	if rollout.Spec.Strategy.Progressive != nil {
		pauseAfterWaveName = rollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName
	}

	for _, wave := range strategy.Spec.Waves {
		waveClusters, err := r.store.ListClusters(ctx, rollout.Spec.Targets.Selector, wave.Targets.Selector)
		if err != nil {
			return err
		}

		if len(waveClusters.Items) == 0 {
			return fmt.Errorf("wave %q does not target any clusters", wave.Name)
		}

		for _, cluster := range waveClusters.Items {
			currentClusterWave, found := clusterWaveMap[cluster.Name]
			if !found {
				// this should never happen
				return fmt.Errorf("wave %q references cluster %s not selected by the rollout", wave.Name, cluster.Name)
			}

			if currentClusterWave != "" {
				return fmt.Errorf("a cluster cannot be selected by more than one wave - cluster %s is selected by waves %q and %q", cluster.Name, currentClusterWave, wave.Name)
			}

			clusterWaveMap[cluster.Name] = wave.Name
		}

		pauseWaveNameFound = pauseWaveNameFound || pauseAfterWaveName == wave.Name
	}

	for _, cluster := range allClusters.Items {
		wave, _ := clusterWaveMap[cluster.Name]
		if wave == "" {
			return fmt.Errorf("waves should cover all clusters selected by the rollout - cluster %s is not covered by any waves", cluster.Name)
		}
	}

	if pauseAfterWaveName != "" && !pauseWaveNameFound {
		return fmt.Errorf("%q pause wave not found in progressive rollout strategy", pauseAfterWaveName)
	}

	return nil
}

func (r *RolloutReconciler) getPackageDiscoveryClient(rolloutNamespacedName types.NamespacedName) *packagediscovery.PackageDiscovery {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	client, found := r.packageDiscoveryCache[rolloutNamespacedName]
	if !found {
		client = packagediscovery.NewPackageDiscovery(r.Client, rolloutNamespacedName.Namespace)
		r.packageDiscoveryCache[rolloutNamespacedName] = client
	}

	return client
}

func (r *RolloutReconciler) reconcileRollout(ctx context.Context, rollout *gitopsv1alpha1.Rollout, strategy *gitopsv1alpha1.ProgressiveRolloutStrategy, packageDiscoveryClient *packagediscovery.PackageDiscovery) error {
	logger := log.FromContext(ctx)

	discoveredPackages, err := packageDiscoveryClient.GetPackages(ctx, rollout.Spec.Packages)
	if err != nil {
		logger.Error(err, "failed to discover packages")
		return client.IgnoreNotFound(err)
	}
	logger.Info("discovered packages", "count", len(discoveredPackages), "packages", discoveredPackages)

	allClusterStatuses := []gitopsv1alpha1.ClusterStatus{}

	pauseFutureWaves := false
	pauseAfterWaveName := ""
	afterPauseAfterWave := false

	if rollout.Spec.Strategy.Progressive != nil {
		pauseAfterWaveName = rollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName
	}

	waveStatuses := []gitopsv1alpha1.WaveStatus{}

	for _, wave := range strategy.Spec.Waves {
		waveClusters, err := r.store.ListClusters(ctx, rollout.Spec.Targets.Selector, wave.Targets.Selector)
		if err != nil {
			return err
		}

		packageClusterMatcherClient := packageclustermatcher.NewPackageClusterMatcher(waveClusters.Items, discoveredPackages)
		allClusterPackages, err := packageClusterMatcherClient.GetClusterPackages(rollout.Spec.PackageToTargetMatcher)
		if err != nil {
			logger.Error(err, "get cluster packages failed")
			return client.IgnoreNotFound(err)
		}

		for _, clusterPackages := range allClusterPackages {
			clusterName := clusterPackages.Cluster.Name
			logger.Info("cluster packages", "cluster", clusterName, "packagesCount", len(clusterPackages.Packages), "packages", clusterPackages.Packages)
		}

		targets, err := r.computeTargets(ctx, rollout, allClusterPackages, waveClusters.Items)
		if err != nil {
			return err
		}

		thisWaveInProgress, clusterStatuses, err := r.rolloutTargets(ctx, rollout, &wave, targets, pauseFutureWaves)
		if err != nil {
			return err
		}

		if thisWaveInProgress {
			pauseFutureWaves = true
		}

		allClusterStatuses = append(allClusterStatuses, clusterStatuses...)

		waveStatuses = append(waveStatuses, getWaveStatus(wave, clusterStatuses, afterPauseAfterWave))

		if wave.Name == pauseAfterWaveName {
			pauseFutureWaves = true
			afterPauseAfterWave = true
		}
	}

	sortClusterStatuses(allClusterStatuses)

	if err := r.updateStatus(ctx, rollout, waveStatuses, allClusterStatuses); err != nil {
		return err
	}
	return nil
}

func (r *RolloutReconciler) updateStatus(ctx context.Context, rollout *gitopsv1alpha1.Rollout, waveStatuses []gitopsv1alpha1.WaveStatus, clusterStatuses []gitopsv1alpha1.ClusterStatus) error {
	logger := log.FromContext(ctx)
	logger.Info("updating the status", "cluster statuses", len(clusterStatuses))

	rollout.Status.Overall = getOverallStatus(clusterStatuses)

	if len(waveStatuses) > 1 {
		rollout.Status.WaveStatuses = waveStatuses
	}

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
	clusterPackages []packageclustermatcher.ClusterPackages, allowClusters []gkeclusterapis.ContainerCluster) (*Targets, error) {

	RRSkeysToBeDeleted := map[client.ObjectKey]*gitopsv1alpha1.RemoteRootSync{}
	// let's take a look at existing remoterootsyncs
	existingRRSs, err := r.listRemoteRootSyncs(ctx, rollout.Name, rollout.Namespace)
	if err != nil {
		return nil, err
	}

	// initially assume all the keys to be deleted
	for _, rrs := range existingRRSs {
		for _, cluster := range allowClusters {
			if rrs.Spec.ClusterRef.Name == cluster.Name {
				RRSkeysToBeDeleted[client.ObjectKeyFromObject(rrs)] = rrs
			}
		}
	}

	klog.Infof("Found remoterootsyncs: %s", toRemoteRootSyncNames(existingRRSs))
	targets := &Targets{}
	// track keys of all the desired remote rootsyncs
	for idx, clusterPkg := range clusterPackages {
		// TODO: figure out multiple packages per cluster story
		if len(clusterPkg.Packages) < 1 {
			continue
		}
		cluster := &clusterPackages[idx].Cluster
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

func (r *RolloutReconciler) rolloutTargets(ctx context.Context, rollout *gitopsv1alpha1.Rollout, wave *gitopsv1alpha1.Wave, targets *Targets, pauseWave bool) (bool, []gitopsv1alpha1.ClusterStatus, error) {
	clusterStatuses := []gitopsv1alpha1.ClusterStatus{}

	concurrentUpdates := 0
	maxConcurrent := int(wave.MaxConcurrent)
	waiting := "Waiting"

	if pauseWave {
		maxConcurrent = 0
		waiting = "Waiting (Upcoming Wave)"
	}

	for _, target := range targets.Unchanged {
		if !isRRSSynced(target) {
			concurrentUpdates++
		}
	}

	for _, target := range targets.ToBeCreated {
		rootSyncSpec := toRootSyncSpec(target.packageRef)
		rrs := newRemoteRootSync(rollout,
			gitopsv1alpha1.ClusterRef{Name: target.cluster.Name},
			rootSyncSpec,
			pkgID(target.packageRef),
		)

		if maxConcurrent > concurrentUpdates {
			if err := r.Create(ctx, rrs); err != nil {
				klog.Warningf("Error creating RemoteRootSync %s: %v", rrs.Name, err)
				return false, nil, err
			}
			concurrentUpdates++
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: rrs.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  rrs.Name,
					SyncStatus: rrs.Status.SyncStatus,
					Status:     "Progressing",
				},
			})
		} else {
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: rrs.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  rrs.Name,
					SyncStatus: "",
					Status:     waiting,
				},
			})
		}
	}

	for _, target := range targets.ToBeUpdated {
		if maxConcurrent > concurrentUpdates {
			if err := r.Update(ctx, target); err != nil {
				klog.Warningf("Error updating RemoteRootSync %s: %v", target.Name, err)
				return false, nil, err
			}
			concurrentUpdates++
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: target.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  target.Name,
					SyncStatus: target.Status.SyncStatus,
					Status:     "Progressing",
				},
			})
		} else {
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: target.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  target.Name,
					SyncStatus: "OutOfSync",
					Status:     waiting,
				},
			})
		}
	}

	for _, target := range targets.ToBeDeleted {
		if maxConcurrent > concurrentUpdates {
			if err := r.Delete(ctx, target); err != nil {
				klog.Warningf("Error deleting RemoteRootSync %s: %v", target.Name, err)
				return false, nil, err
			}
			concurrentUpdates++
		} else {
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: target.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  target.Name,
					SyncStatus: "OutOfSync",
					Status:     waiting,
				},
			})
		}
	}

	for _, target := range targets.Unchanged {
		status := "Progressing"

		if isRRSSynced(target) {
			status = "Synced"
		} else if isRRSErrored(target) {
			status = "Stalled"
		}

		clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
			Name: target.Spec.ClusterRef.Name,
			PackageStatus: gitopsv1alpha1.PackageStatus{
				PackageID:  target.Name,
				SyncStatus: target.Status.SyncStatus,
				Status:     status,
			},
		})
	}

	thisWaveInProgress := concurrentUpdates > 0

	sortClusterStatuses(clusterStatuses)

	return thisWaveInProgress, clusterStatuses, nil
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

func (r *RolloutReconciler) listAllRollouts(ctx context.Context) ([]*gitopsv1alpha1.Rollout, error) {
	var list gitopsv1alpha1.RolloutList
	if err := r.List(ctx, &list); err != nil {
		return nil, err
	}
	var rollouts []*gitopsv1alpha1.Rollout
	for i := range list.Items {
		item := &list.Items[i]
		rollouts = append(rollouts, item)
	}
	return rollouts, nil
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

func isRRSErrored(rss *gitopsv1alpha1.RemoteRootSync) bool {
	if rss.Generation != rss.Status.ObservedGeneration {
		return false
	}

	if rss.Status.SyncStatus == "Error" {
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
	if dpkg.Directory == "" || dpkg.Directory == "." || dpkg.Directory == "/" {
		return fmt.Sprintf("%s-%s", dpkg.Org, dpkg.Repo)
	}

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

	var containerCluster unstructured.Unstructured
	containerCluster.SetGroupVersionKind(kccClusterGVK)

	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1alpha1.Rollout{}).
		Owns(&gitopsv1alpha1.RemoteRootSync{}).
		Watches(
			&source.Kind{Type: &containerCluster},
			handler.EnqueueRequestsFromMapFunc(r.mapClusterUpdateToRequest),
		).
		Complete(r)
}

func (r *RolloutReconciler) mapClusterUpdateToRequest(cluster client.Object) []reconcile.Request {
	logger := log.FromContext(context.Background())

	var requests []reconcile.Request

	allRollouts, err := r.listAllRollouts(context.Background())
	if err != nil {
		logger.Error(err, "failed to list rollouts")
		return []reconcile.Request{}
	}

	for _, rollout := range allRollouts {
		selector, err := metav1.LabelSelectorAsSelector(rollout.Spec.Targets.Selector)
		if err != nil {
			logger.Error(err, "failed to create label selector", "rolloutName", rollout.Name)
			continue
		}

		rolloutDeploysToCluster := rolloutIncludesCluster(rollout, cluster.GetName())
		clusterInTargetSet := selector.Matches(labels.Set(cluster.GetLabels()))

		if rolloutDeploysToCluster || clusterInTargetSet {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: rollout.Name, Namespace: rollout.Namespace}})
		}
	}

	return requests
}

func sortClusterStatuses(clusterStatuses []gitopsv1alpha1.ClusterStatus) {
	sort.Slice(clusterStatuses, func(i, j int) bool {
		return strings.Compare(clusterStatuses[i].Name, clusterStatuses[j].Name) == -1
	})
}

func getWaveStatus(wave gitopsv1alpha1.Wave, clusterStatuses []gitopsv1alpha1.ClusterStatus, wavePaused bool) gitopsv1alpha1.WaveStatus {
	return gitopsv1alpha1.WaveStatus{
		Name:            wave.Name,
		Status:          getOverallStatus(clusterStatuses),
		Paused:          wavePaused,
		ClusterStatuses: clusterStatuses,
	}
}

func getOverallStatus(clusterStatuses []gitopsv1alpha1.ClusterStatus) string {
	overall := "Completed"

	anyProgressing := false
	anyStalled := false
	anyWaiting := false

	for _, clusterStatus := range clusterStatuses {
		switch {
		case clusterStatus.PackageStatus.Status == "Progressing":
			anyProgressing = true

		case clusterStatus.PackageStatus.Status == "Stalled":
			anyStalled = true

		case strings.HasPrefix(clusterStatus.PackageStatus.Status, "Waiting"):
			anyWaiting = true
		}
	}

	switch {
	case anyProgressing:
		overall = "Progressing"
	case anyStalled:
		overall = "Stalled"
	case anyWaiting:
		overall = "Waiting"
	}

	return overall
}

func rolloutIncludesCluster(rollout *gitopsv1alpha1.Rollout, clusterName string) bool {
	for _, clusterStatus := range rollout.Status.ClusterStatuses {
		if clusterStatus.Name == clusterName {
			return true
		}
	}

	return false
}
