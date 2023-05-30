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
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
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

//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=container.cnrm.cloud.google.com,resources=containerclusters,verbs=get;list;watch
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
// For each kcc cluster, fetch external sync objects in each of the KCC clusters.
func (r *RolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.NewKlogr().WithValues("controller", "rollout", "rollout", req.NamespacedName)
	ctx = klog.NewContext(ctx, logger)

	var rollout gitopsv1alpha1.Rollout

	logger.Info("Reconciling")

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

	targetClusters, err := r.store.ListClusters(ctx, &rollout.Spec.Clusters, rollout.Spec.Targets.Selector)
	if err != nil {
		logger.Error(err, "Failed to list clusters")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	discoveredPackages, err := packageDiscoveryClient.GetPackages(ctx, rollout.Spec.Packages)
	if err != nil {
		logger.Error(err, "Failed to discover packages")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info("Discovered packages", "packagesCount", len(discoveredPackages), "packages", packagediscovery.ToStr(discoveredPackages))

	allClusterStatuses, waveStatuses, err := r.reconcileRollout(ctx, &rollout, strategy, targetClusters, discoveredPackages)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, &rollout, waveStatuses, allClusterStatuses); err != nil {
		return ctrl.Result{}, err
	}

	if rollout.Spec.Clusters.SourceType == gitopsv1alpha1.GCPFleet &&
		(rollout.Status.Overall == "Completed" || rollout.Status.Overall == "Stalled") {
		// TODO (droot): The rollouts in completed/stalled state will not be reconciled
		// whenever fleet memberships change, so scheduling a periodic reconcile
		// until we fix https://github.com/GoogleContainerTools/kpt/issues/3835
		// This can be safely removed once we start monitoring fleet changes.
		// Note: we watch containercluster types, so this problem doesn't exist for the
		// KCC clusters.
		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *RolloutReconciler) getStrategy(ctx context.Context, rollout *gitopsv1alpha1.Rollout) (*gitopsv1alpha1.ProgressiveRolloutStrategy, error) {
	logger := klog.FromContext(ctx)

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
			logger.Error(err, "Unable to fetch progressive rollout strategy", "strategyRef", strategyRef)
			// TODO (droot): signal this as a condition in the rollout status
			return nil, err
		}

		err := r.validateProgressiveRolloutStrategy(ctx, rollout, &progressiveStrategy)
		if err != nil {
			logger.Error(err, "Progressive rollout strategy failed validation", "strategyRef", strategyRef)
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
	allClusters, err := r.store.ListClusters(ctx, &rollout.Spec.Clusters, rollout.Spec.Targets.Selector)
	if err != nil {
		return err
	}

	clusterWaveMap := make(map[string]string)
	for _, cluster := range allClusters {
		clusterWaveMap[cluster.Ref.Name] = ""
	}

	pauseAfterWaveName := ""
	pauseWaveNameFound := false

	if rollout.Spec.Strategy.Progressive != nil {
		pauseAfterWaveName = rollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName
	}

	for _, wave := range strategy.Spec.Waves {
		waveClusters, err := filterClusters(allClusters, wave.Targets.Selector)
		if err != nil {
			return err
		}

		if len(waveClusters) == 0 {
			return fmt.Errorf("wave %q does not target any clusters", wave.Name)
		}

		for _, cluster := range waveClusters {
			currentClusterWave, found := clusterWaveMap[cluster.Ref.Name]
			if !found {
				// this should never happen
				return fmt.Errorf("wave %q references cluster %s not selected by the rollout", wave.Name, cluster.Ref.Name)
			}

			if currentClusterWave != "" {
				return fmt.Errorf("a cluster cannot be selected by more than one wave - cluster %s is selected by waves %q and %q", cluster.Ref.Name, currentClusterWave, wave.Name)
			}

			clusterWaveMap[cluster.Ref.Name] = wave.Name
		}

		pauseWaveNameFound = pauseWaveNameFound || pauseAfterWaveName == wave.Name
	}

	for _, cluster := range allClusters {
		wave, _ := clusterWaveMap[cluster.Ref.Name]
		if wave == "" {
			return fmt.Errorf("waves should cover all clusters selected by the rollout - cluster %s is not covered by any waves", cluster.Ref.Name)
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

func (r *RolloutReconciler) reconcileRollout(ctx context.Context, rollout *gitopsv1alpha1.Rollout, strategy *gitopsv1alpha1.ProgressiveRolloutStrategy, targetClusters []clusterstore.Cluster,
	discoveredPackages []packagediscovery.DiscoveredPackage) ([]gitopsv1alpha1.ClusterStatus, []gitopsv1alpha1.WaveStatus, error) {

	packageClusterMatcherClient := packageclustermatcher.NewPackageClusterMatcher(targetClusters, discoveredPackages)
	clusterPackages, err := packageClusterMatcherClient.GetClusterPackages(rollout.Spec.PackageToTargetMatcher)
	if err != nil {
		return nil, nil, err
	}

	targets, err := r.computeTargets(ctx, rollout, clusterPackages)
	if err != nil {
		return nil, nil, err
	}

	allClusterStatuses := []gitopsv1alpha1.ClusterStatus{}
	waveStatuses := []gitopsv1alpha1.WaveStatus{}

	allWaveTargets, err := r.getWaveTargets(ctx, rollout, targets, targetClusters, strategy.Spec.Waves)
	if err != nil {
		return nil, nil, err
	}

	pauseFutureWaves := false
	pauseAfterWaveName := ""
	afterPauseAfterWave := false

	if rollout.Spec.Strategy.Progressive != nil {
		pauseAfterWaveName = rollout.Spec.Strategy.Progressive.PauseAfterWave.WaveName
	}

	for i := range allWaveTargets {
		thisWaveTargets := allWaveTargets[i]
		waveTargets := thisWaveTargets.Targets
		wave := thisWaveTargets.Wave

		thisWaveInProgress, clusterStatuses, err := r.rolloutTargets(ctx, rollout, wave, waveTargets, pauseFutureWaves)
		if err != nil {
			return nil, nil, err
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

	return allClusterStatuses, waveStatuses, nil
}

func (r *RolloutReconciler) updateStatus(ctx context.Context, rollout *gitopsv1alpha1.Rollout, waveStatuses []gitopsv1alpha1.WaveStatus, clusterStatuses []gitopsv1alpha1.ClusterStatus) error {
	logger := klog.FromContext(ctx)
	logger.Info("Updating status", "clusterStatusesCount", len(clusterStatuses), "waveStatusesCount", len(waveStatuses))

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
compute the RemoteSync corresponding to each (cluster, package) pair
For RemoteSync, name has to be function of cluster-id and package-id.
For RemoteSync, make rootSyncTemplate
*/
func (r *RolloutReconciler) computeTargets(ctx context.Context,
	rollout *gitopsv1alpha1.Rollout,
	clusterPackages []packageclustermatcher.ClusterPackages) (*Targets, error) {
	logger := klog.FromContext(ctx)

	RSkeysToBeDeleted := map[client.ObjectKey]*gitopsv1alpha1.RemoteSync{}
	// let's take a look at existing remotesyncs
	existingRSs, err := r.listRemoteSyncs(ctx, rollout.Name, rollout.Namespace)
	if err != nil {
		return nil, err
	}

	// initially assume all the keys to be deleted
	for _, rs := range existingRSs {
		RSkeysToBeDeleted[client.ObjectKeyFromObject(rs)] = rs
	}

	logger.Info("Found RemoteSyncs", "remoteSyncs", toRemoteSyncNames(existingRSs))
	targets := &Targets{}
	// track keys of all the desired remote syncs
	for idx, clusterPkg := range clusterPackages {
		// TODO: figure out multiple packages per cluster story
		if len(clusterPkg.Packages) < 1 {
			continue
		}
		cluster := &clusterPackages[idx].Cluster
		clusterName := cluster.Ref.Name[strings.LastIndex(cluster.Ref.Name, "/")+1:]
		pkg := &clusterPkg.Packages[0]
		rs := gitopsv1alpha1.RemoteSync{}
		key := client.ObjectKey{
			Namespace: rollout.Namespace,
			Name:      makeRemoteSyncName(clusterName, rollout.GetName()),
		}
		// since this RS need to exist, remove it from the deletion list
		delete(RSkeysToBeDeleted, key)
		// check if this remotesync for this package exists or not ?
		err := r.Client.Get(ctx, key, &rs)
		if err != nil {
			if apierrors.IsNotFound(err) { // rs is missing
				targets.ToBeCreated = append(targets.ToBeCreated, &clusterPackagePair{
					cluster:    cluster,
					packageRef: pkg,
				})
			} else {
				// some other error encountered
				return nil, err
			}
		} else {
			// remotesync already exists
			updated, needsUpdate := rsNeedsUpdate(ctx, rollout, &rs, &clusterPackagePair{
				cluster:    cluster,
				packageRef: pkg,
			})
			if needsUpdate {
				targets.ToBeUpdated = append(targets.ToBeUpdated, updated)
			} else {
				targets.Unchanged = append(targets.Unchanged, &rs)
			}
		}
	}

	for _, rs := range RSkeysToBeDeleted {
		targets.ToBeDeleted = append(targets.ToBeDeleted, rs)
	}

	return targets, nil
}

// rsNeedsUpdate checks if the underlying remotesync needs to be updated by creating a new RemoteSync object and comparing it to the existing one
func rsNeedsUpdate(ctx context.Context, rollout *gitopsv1alpha1.Rollout, currentRS *gitopsv1alpha1.RemoteSync, target *clusterPackagePair) (*gitopsv1alpha1.RemoteSync, bool) {
	desiredRS := newRemoteSync(rollout, target)

	// if the spec of the new RemoteSync object is not identical to the existing one, then an update is necessary
	if !equality.Semantic.DeepEqual(currentRS.Spec, desiredRS.Spec) {
		currentRS.Spec = desiredRS.Spec
		return currentRS, true
	}

	// no update required, return nil
	return nil, false
}

func (r *RolloutReconciler) getWaveTargets(ctx context.Context, rollout *gitopsv1alpha1.Rollout, allTargets *Targets, allClusters []clusterstore.Cluster,
	allWaves []gitopsv1alpha1.Wave) ([]WaveTarget, error) {
	allWaveTargets := []WaveTarget{}

	clusterNameToWaveTarget := make(map[string]*WaveTarget)

	for i := range allWaves {
		wave := allWaves[i]
		thisWaveTarget := WaveTarget{Wave: &wave, Targets: &Targets{}}

		waveClusters, err := filterClusters(allClusters, wave.Targets.Selector)
		if err != nil {
			return nil, err
		}

		for _, cluster := range waveClusters {
			clusterNameToWaveTarget[cluster.Ref.Name] = &thisWaveTarget
		}

		allWaveTargets = append(allWaveTargets, thisWaveTarget)
	}

	for _, toCreate := range allTargets.ToBeCreated {
		wavetTargets := clusterNameToWaveTarget[toCreate.cluster.Ref.Name].Targets
		wavetTargets.ToBeCreated = append(wavetTargets.ToBeCreated, toCreate)
	}

	for _, rs := range allTargets.ToBeUpdated {
		wavetTargets := clusterNameToWaveTarget[rs.Spec.ClusterRef.Name].Targets
		wavetTargets.ToBeUpdated = append(wavetTargets.ToBeUpdated, rs)
	}

	for _, rs := range allTargets.Unchanged {
		wavetTargets := clusterNameToWaveTarget[rs.Spec.ClusterRef.Name].Targets
		wavetTargets.Unchanged = append(wavetTargets.Unchanged, rs)
	}

	for _, rs := range allTargets.ToBeDeleted {
		// The remote sync will be associated back to it's previous wave and then removed as part
		// of that wave. If the previous wave the remote sync cannot be determined, then the remote
		// sync will be removed with the last wave of the rollout.

		waveName, found := findWaveNameForCluster(rollout, rs.Spec.ClusterRef.Name)

		if !found {
			waveName = allWaveTargets[len(allWaveTargets)-1].Wave.Name
		}

		for _, waveTarget := range allWaveTargets {
			if waveTarget.Wave.Name == waveName {
				wavetTargets := waveTarget.Targets
				wavetTargets.ToBeDeleted = append(wavetTargets.ToBeDeleted, rs)
			}
		}
	}

	return allWaveTargets, nil
}

func (r *RolloutReconciler) rolloutTargets(ctx context.Context, rollout *gitopsv1alpha1.Rollout, wave *gitopsv1alpha1.Wave, targets *Targets, pauseWave bool) (bool, []gitopsv1alpha1.ClusterStatus, error) {
	clusterStatuses := []gitopsv1alpha1.ClusterStatus{}
	logger := klog.FromContext(ctx)

	concurrentUpdates := 0
	maxConcurrent := int(wave.MaxConcurrent)
	waiting := "Waiting"

	if pauseWave {
		maxConcurrent = 0
		waiting = "Waiting (Upcoming Wave)"
	}

	for _, target := range targets.Unchanged {
		if !isRSSynced(target) {
			concurrentUpdates++
		}
	}

	for _, target := range targets.ToBeCreated {
		rs := newRemoteSync(rollout, target)
		if maxConcurrent > concurrentUpdates {
			if err := r.Create(ctx, rs); err != nil {
				logger.Info("Warning, error creating RemoteSync", "remoteSync", klog.KRef(rs.Namespace, rs.Name), "err", err)
				return false, nil, err
			}
			concurrentUpdates++
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: rs.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  rs.Name,
					SyncStatus: rs.Status.SyncStatus,
					Status:     "Progressing",
				},
			})
		} else {
			clusterStatuses = append(clusterStatuses, gitopsv1alpha1.ClusterStatus{
				Name: rs.Spec.ClusterRef.Name,
				PackageStatus: gitopsv1alpha1.PackageStatus{
					PackageID:  rs.Name,
					SyncStatus: "",
					Status:     waiting,
				},
			})
		}
	}

	for _, target := range targets.ToBeUpdated {
		if maxConcurrent > concurrentUpdates {
			if err := r.Update(ctx, target); err != nil {
				logger.Info("Warning, cannot update RemoteSync", "remoteSync", klog.KRef(target.Namespace, target.Name), "err", err)
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
				logger.Info("Warning, cannot delete RemoteSync", "remoteSync", klog.KRef(target.Namespace, target.Name), "err", err)
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

		if isRSSynced(target) {
			status = "Synced"
		} else if isRSErrored(target) {
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

type WaveTarget struct {
	Wave    *gitopsv1alpha1.Wave
	Targets *Targets
}

type Targets struct {
	ToBeCreated []*clusterPackagePair
	ToBeUpdated []*gitopsv1alpha1.RemoteSync
	ToBeDeleted []*gitopsv1alpha1.RemoteSync
	Unchanged   []*gitopsv1alpha1.RemoteSync
}

type clusterPackagePair struct {
	cluster    *clusterstore.Cluster
	packageRef *packagediscovery.DiscoveredPackage
}

func toRemoteSyncNames(rsss []*gitopsv1alpha1.RemoteSync) []string {
	var names []string
	for _, rss := range rsss {
		names = append(names, rss.Name)
	}
	return names
}

func (r *RolloutReconciler) listRemoteSyncs(ctx context.Context, rsdName, rsdNamespace string) ([]*gitopsv1alpha1.RemoteSync, error) {
	var list gitopsv1alpha1.RemoteSyncList
	if err := r.List(ctx, &list, client.MatchingLabels{rolloutLabel: rsdName}, client.InNamespace(rsdNamespace)); err != nil {
		return nil, err
	}
	var remotesyncs []*gitopsv1alpha1.RemoteSync
	for i := range list.Items {
		item := &list.Items[i]
		remotesyncs = append(remotesyncs, item)
	}
	return remotesyncs, nil
}

func (r *RolloutReconciler) listAllRollouts(ctx context.Context) ([]gitopsv1alpha1.Rollout, error) {
	var rolloutsList gitopsv1alpha1.RolloutList
	if err := r.List(ctx, &rolloutsList); err != nil {
		return nil, err
	}

	return rolloutsList.Items, nil
}

func isRSSynced(rss *gitopsv1alpha1.RemoteSync) bool {
	if rss.Generation != rss.Status.ObservedGeneration {
		return false
	}

	if rss.Status.SyncStatus == "Synced" {
		return true
	}
	return false
}

func isRSErrored(rss *gitopsv1alpha1.RemoteSync) bool {
	if rss.Generation != rss.Status.ObservedGeneration {
		return false
	}

	if rss.Status.SyncStatus == "Error" {
		return true
	}
	return false
}

// Given a package identifier and cluster, create a RemoteSync object.
func newRemoteSync(rollout *gitopsv1alpha1.Rollout, target *clusterPackagePair) *gitopsv1alpha1.RemoteSync {
	t := true
	clusterRef := target.cluster.Ref
	clusterName := clusterRef.Name[strings.LastIndex(clusterRef.Name, "/")+1:]

	templateType := gitopsv1alpha1.TemplateTypeRootSync
	if rollout != nil && rollout.Spec.SyncTemplate != nil {
		templateType = rollout.Spec.SyncTemplate.Type
	}

	// The RemoteSync object is created in the same namespace as the Rollout
	// object. The RemoteSync will create either a RepoSync in the same namespace,
	// or a RootSync in the config-management-system namespace.
	return &gitopsv1alpha1.RemoteSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeRemoteSyncName(clusterName, rollout.GetName()),
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

		Spec: gitopsv1alpha1.RemoteSyncSpec{
			Type:       templateType,
			ClusterRef: clusterRef,
			Template: &gitopsv1alpha1.Template{
				Spec:     toSyncSpec(target.packageRef, rollout),
				Metadata: getSpecMetadata(rollout),
			},
		},
	}
}

func toSyncSpec(dpkg *packagediscovery.DiscoveredPackage, rollout *gitopsv1alpha1.Rollout) *gitopsv1alpha1.SyncSpec {
	syncSpec := &gitopsv1alpha1.SyncSpec{
		SourceFormat: "unstructured",
	}
	switch {
	case dpkg.OciRepo != nil:
		syncSpec.SourceType = "oci"
		syncSpec.Oci = &gitopsv1alpha1.OciInfo{
			Image: dpkg.OciRepo.Image,
			Dir:   dpkg.Directory,
		}
		// copy the fields from the RSync template
		if rollout.Spec.SyncTemplate.RepoSync != nil {
			syncSpec.Oci.Auth = rollout.Spec.SyncTemplate.RepoSync.Oci.Auth
			syncSpec.Oci.GCPServiceAccountEmail = rollout.Spec.SyncTemplate.RepoSync.Oci.GCPServiceAccountEmail
		} else {
			syncSpec.Oci.Auth = rollout.Spec.SyncTemplate.RootSync.Oci.Auth
			syncSpec.Oci.GCPServiceAccountEmail = rollout.Spec.SyncTemplate.RootSync.Oci.GCPServiceAccountEmail
		}
	default:
		syncSpec.SourceType = "git"
		syncSpec.Git = &gitopsv1alpha1.GitInfo{
			// TODO(droot): Repo URL can be an HTTP, GIT or SSH based URL
			// Need to make it configurable
			Repo:     dpkg.HTTPURL(),
			Revision: dpkg.Revision,
			Dir:      dpkg.Directory,
			Branch:   dpkg.Branch,
			Auth:     "none",
		}
	}
	return syncSpec
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
	clusterStore, err := clusterstore.NewClusterStore(r.Client, mgr.GetConfig())
	if err != nil {
		return err
	}
	r.store = clusterStore

	var containerCluster gkeclusterapis.ContainerCluster
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1alpha1.Rollout{}).
		Owns(&gitopsv1alpha1.RemoteSync{}).
		Watches(
			&source.Kind{Type: &containerCluster},
			handler.EnqueueRequestsFromMapFunc(r.mapClusterUpdateToRequest),
		).
		Complete(r)
}

func (r *RolloutReconciler) mapClusterUpdateToRequest(cluster client.Object) []reconcile.Request {
	logger := klog.FromContext(context.Background())

	var requests []reconcile.Request

	allRollouts, err := r.listAllRollouts(context.Background())
	if err != nil {
		logger.Error(err, "Failed to list rollouts")
		return []reconcile.Request{}
	}

	for _, rollout := range allRollouts {
		selector, err := metav1.LabelSelectorAsSelector(rollout.Spec.Targets.Selector)
		if err != nil {
			logger.Error(err, "Failed to create label selector")
			continue
		}

		rolloutDeploysToCluster := rolloutIncludesCluster(&rollout, cluster.GetName())
		clusterInTargetSet := selector.Matches(labels.Set(cluster.GetLabels()))

		// Rollouts will be reconciled for cluster updates when
		// 1) a cluster is added to the rollout target set (clusterInTargetSet will be true)
		// 2) a cluster is removed from the rollout target saet (rolloutDeploysToCluster will be true)
		// 3) an cluster in the rollout target set is being updated where the package matching logic may produce different results (both variables will be true)
		reconcileRollout := rolloutDeploysToCluster || clusterInTargetSet

		if reconcileRollout {
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

func getWaveStatus(wave *gitopsv1alpha1.Wave, clusterStatuses []gitopsv1alpha1.ClusterStatus, wavePaused bool) gitopsv1alpha1.WaveStatus {
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

func findWaveNameForCluster(rollout *gitopsv1alpha1.Rollout, clusterName string) (string, bool) {
	for _, waveStatus := range rollout.Status.WaveStatuses {
		for _, clusterStatus := range waveStatus.ClusterStatuses {
			if clusterStatus.Name == clusterName {
				return waveStatus.Name, true
			}
		}
	}

	return "", false
}

func rolloutIncludesCluster(rollout *gitopsv1alpha1.Rollout, clusterName string) bool {
	for _, clusterStatus := range rollout.Status.ClusterStatuses {
		if clusterStatus.Name == clusterName {
			return true
		}
	}

	return false
}

func filterClusters(allClusters []clusterstore.Cluster, labelSelector *metav1.LabelSelector) ([]clusterstore.Cluster, error) {
	clusters := []clusterstore.Cluster{}

	for _, cluster := range allClusters {
		clusterLabelSet := labels.Set(cluster.Labels)
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, err
		}

		if selector.Matches(clusterLabelSet) {
			clusters = append(clusters, cluster)
		}
	}

	return clusters, nil
}

func getSpecMetadata(rollout *gitopsv1alpha1.Rollout) *gitopsv1alpha1.Metadata {
	if rollout == nil || rollout.Spec.SyncTemplate == nil {
		return nil
	}
	switch rollout.Spec.SyncTemplate.Type {
	case gitopsv1alpha1.TemplateTypeRepoSync:
		if rollout.Spec.SyncTemplate.RepoSync == nil {
			return nil
		}
		return rollout.Spec.SyncTemplate.RepoSync.Metadata
	case gitopsv1alpha1.TemplateTypeRootSync, "":
		if rollout.Spec.SyncTemplate.RootSync == nil {
			return nil
		}
		return rollout.Spec.SyncTemplate.RootSync.Metadata
	}
	return nil
}
