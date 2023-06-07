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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/clusterstore"
)

var (
	// The RootSync object always gets put in the config-management-system namespace,
	// while the RepoSync object will get its namespace from the RemoteSync object's
	// metadata.namespace.
	// See examples at: https://cloud.google.com/anthos-config-management/docs/how-to/multiple-repositories
	rootSyncNamespace = "config-management-system"

	rootSyncGVK = schema.GroupVersionKind{
		Group:   "configsync.gke.io",
		Version: "v1beta1",
		Kind:    "RootSync",
	}
	rootSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "rootsyncs",
	}

	repoSyncGVK = schema.GroupVersionKind{
		Group:   "configsync.gke.io",
		Version: "v1beta1",
		Kind:    "RepoSync",
	}
	repoSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "reposyncs",
	}

	remoteSyncNameLabel      = "gitops.kpt.dev/remotesync-name"
	remoteSyncNamespaceLabel = "gitops.kpt.dev/remotesync-namespace"
)

const (
	conditionReconciling = "Reconciling"
	conditionStalled     = "Stalled"

	reasonCreateSync = "CreateSync"
	reasonUpdateSync = "UpdateSync"
	reasonError      = "Error"
)

// RemoteSyncReconciler reconciles a RemoteSync object
type RemoteSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	store *clusterstore.ClusterStore

	// channel is where watchers put events to trigger new reconcilations based
	// on watch events from target clusters.
	channel chan event.GenericEvent

	mutex sync.Mutex

	watchers map[gitopsv1alpha1.ClusterRef]*watcher
}

//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remotesyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remotesyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remotesyncs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RemoteSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *RemoteSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.NewKlogr().WithValues("controller", "remotesync", "remoteSync", req.NamespacedName)
	ctx = klog.NewContext(ctx, logger)

	logger.Info("Reconciling")

	var remotesync gitopsv1alpha1.RemoteSync
	if err := r.Get(ctx, req.NamespacedName, &remotesync); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	myFinalizerName := "remotesync.gitops.kpt.dev/finalizer"
	if remotesync.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&remotesync, myFinalizerName) {
			controllerutil.AddFinalizer(&remotesync, myFinalizerName)
			if err := r.Update(ctx, &remotesync); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&remotesync, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if remotesync.Status.SyncCreated {
				// Delete the external sync resource
				err := r.deleteExternalResources(ctx, &remotesync)
				if err != nil && !apierrors.IsNotFound(err) {
					statusError := r.updateStatus(ctx, &remotesync, "", err)

					if statusError != nil {
						logger.Error(statusError, "Failed to update status")
					}

					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return ctrl.Result{}, fmt.Errorf("have problem to delete external resource: %w", err)
				}

				// Make sure we stop any watches that are no longer needed.
				logger.Info("Pruning watches")
				r.pruneWatches(req.NamespacedName, &remotesync.Spec.ClusterRef)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&remotesync, myFinalizerName)
			if err := r.Update(ctx, &remotesync); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	syncStatus, syncError := r.syncExternalSync(ctx, &remotesync)

	if err := r.updateStatus(ctx, &remotesync, syncStatus, syncError); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, syncError
}

func (r *RemoteSyncReconciler) syncExternalSync(ctx context.Context, rs *gitopsv1alpha1.RemoteSync) (string, error) {
	clusterRef := &rs.Spec.ClusterRef

	dynCl, err := r.getDynamicClientForCluster(ctx, clusterRef)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	if err := r.patchExternalSync(ctx, dynCl, rs); err != nil {
		return "", fmt.Errorf("failed to create/update sync: %w", err)
	}

	r.setupWatches(ctx, getExternalSyncName(rs), rs.Namespace, rs.Spec.ClusterRef)

	syncStatus, err := checkSyncStatus(ctx, dynCl, rs)
	if err != nil {
		return "", fmt.Errorf("failed to check status: %w", err)
	}

	return syncStatus, nil
}

func (r *RemoteSyncReconciler) updateStatus(ctx context.Context, rs *gitopsv1alpha1.RemoteSync, syncStatus string, syncError error) error {
	logger := klog.FromContext(ctx)

	rsPrior := rs.DeepCopy()
	conditions := &rs.Status.Conditions

	if syncError == nil {
		rs.Status.SyncStatus = syncStatus
		rs.Status.SyncCreated = true

		meta.RemoveStatusCondition(conditions, conditionReconciling)
		meta.RemoveStatusCondition(conditions, conditionStalled)
	} else {
		reconcileReason := reasonUpdateSync

		rs.Status.SyncStatus = "Unknown"

		if !rs.Status.SyncCreated {
			rs.Status.SyncStatus = ""
			reconcileReason = reasonCreateSync
		}

		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionReconciling, Status: metav1.ConditionTrue, Reason: reconcileReason})
		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionStalled, Status: metav1.ConditionTrue, Reason: reasonError, Message: syncError.Error()})
	}

	rs.Status.ObservedGeneration = rs.Generation

	if reflect.DeepEqual(rs.Status, rsPrior.Status) {
		return nil
	}

	logger.Info("Updating status")
	return r.Client.Status().Update(ctx, rs)
}

// patchExternalSync patches the external sync in the remote clusters targeted by
// the clusterRefs based on the latest revision of the template in the RemoteSync.
func (r *RemoteSyncReconciler) patchExternalSync(ctx context.Context, client dynamic.Interface, rs *gitopsv1alpha1.RemoteSync) error {
	logger := klog.FromContext(ctx)

	gvr, gvk, err := getGvrAndGvk(rs.Spec.Type)
	if err != nil {
		return err
	}

	namespace := getExternalSyncNamespace(rs)

	newRootSync, err := BuildObjectsToApply(rs, gvk, namespace)
	if err != nil {
		return err
	}
	data, err := json.Marshal(newRootSync)
	if err != nil {
		return fmt.Errorf("failed to encode %s to JSON: %w", gvk.Kind, err)
	}

	_, err = client.Resource(gvr).Namespace(namespace).Patch(ctx, newRootSync.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: rs.Name})
	if err != nil {
		return fmt.Errorf("failed to patch %s: %w", gvk.Kind, err)
	}

	logger.Info(fmt.Sprintf("%s resource created/updated", gvk.Kind), gvr.Resource, klog.KRef(namespace, rs.Name))
	return nil
}

// setupWatches makes sure we have the necessary watches running against
// the remote clusters we care about.
func (r *RemoteSyncReconciler) setupWatches(ctx context.Context, rsName, ns string, clusterRef gitopsv1alpha1.ClusterRef) {
	logger := klog.FromContext(ctx)

	r.mutex.Lock()
	defer r.mutex.Unlock()
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      rsName,
	}

	// If we already have a watch running, make sure we have the current Sync Set
	// listed in the liens map.
	if w, found := r.watchers[clusterRef]; found {
		w.liens[nn] = struct{}{}
		return
	}

	getDynamicClient := func(ctx context.Context) (dynamic.Interface, error) {
		return r.getDynamicClientForCluster(ctx, &clusterRef)
	}

	// Since we don't currently have a watch running, create a new watcher
	// and add it to the map of watchers.
	watcherCtx, cancelFunc := context.WithCancel(context.Background())
	watcherCtx = klog.NewContext(watcherCtx, logger.WithValues("clusterRef", clusterRef.Name))
	w := &watcher{
		clusterRef:       clusterRef,
		ctx:              watcherCtx,
		cancelFunc:       cancelFunc,
		getDynamicClient: getDynamicClient,
		channel:          r.channel,
		liens: map[types.NamespacedName]struct{}{
			nn: {},
		},
	}

	logger.Info("Creating watcher")
	go w.watch()
	r.watchers[clusterRef] = w
}

// pruneWatches removes the current Sync Set from the liens map of all watchers
// that it no longer needs. If any of the watchers are no longer used by any Sync Sets,
// they are shut down.
func (r *RemoteSyncReconciler) pruneWatches(rsnn types.NamespacedName, clusterRef *gitopsv1alpha1.ClusterRef) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Look through all watchers to check if it used to be needed by the Sync Set
	// but is no longer.
	w, found := r.watchers[*clusterRef]
	if !found {
		return
	}

	// Delete the current Sync Set from the list of liens (it it exists)
	delete(w.liens, rsnn)
	// If no other Sync Sets need the watch, stop it and remove the watcher from the map.
	if len(w.liens) == 0 {
		w.cancelFunc()
		delete(r.watchers, *clusterRef)
	}
}

// BuildObjectsToApply configures the external sync
func BuildObjectsToApply(remotesync *gitopsv1alpha1.RemoteSync,
	gvk schema.GroupVersionKind,
	namespace string) (*unstructured.Unstructured, error) {

	newRootSync, err := runtime.DefaultUnstructuredConverter.ToUnstructured(remotesync.Spec.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to unstructured type: %w", err)
	}

	u := unstructured.Unstructured{Object: newRootSync}
	u.SetGroupVersionKind(gvk)
	u.SetName(getExternalSyncName(remotesync))
	u.SetNamespace(namespace)

	labels := u.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[remoteSyncNameLabel] = remotesync.Name
	labels[remoteSyncNamespaceLabel] = remotesync.Namespace
	u.SetLabels(labels)

	return &u, nil
}

func (r *RemoteSyncReconciler) deleteExternalResources(ctx context.Context, remotesync *gitopsv1alpha1.RemoteSync) error {
	logger := klog.FromContext(ctx)

	clusterRef := &remotesync.Spec.ClusterRef
	dynCl, err := r.getDynamicClientForCluster(ctx, clusterRef)
	if err != nil {
		return err
	}

	gvr, _, err := getGvrAndGvk(remotesync.Spec.Type)
	if err != nil {
		return err
	}

	logger.Info("Deleting external resource")
	err = dynCl.Resource(gvr).Namespace(getExternalSyncNamespace(remotesync)).Delete(ctx, getExternalSyncName(remotesync), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	logger.Info("External resource deleted")
	return err
}

func (r *RemoteSyncReconciler) getDynamicClientForCluster(ctx context.Context, clusterRef *gitopsv1alpha1.ClusterRef) (dynamic.Interface, error) {
	restConfig, err := r.store.GetRESTConfig(ctx, clusterRef)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}

func getGvrAndGvk(t gitopsv1alpha1.SyncTemplateType) (schema.GroupVersionResource, schema.GroupVersionKind, error) {
	switch t {
	case gitopsv1alpha1.TemplateTypeRootSync, "":
		return rootSyncGVR, rootSyncGVK, nil
	case gitopsv1alpha1.TemplateTypeRepoSync:
		return repoSyncGVR, repoSyncGVK, nil
	default:
		return schema.GroupVersionResource{}, schema.GroupVersionKind{}, fmt.Errorf("invalid sync type %q", t)
	}
}

func getExternalSyncNamespace(rs *gitopsv1alpha1.RemoteSync) string {
	if rs.Spec.Type == gitopsv1alpha1.TemplateTypeRepoSync {
		return rs.Namespace
	} else {
		return rootSyncNamespace
	}
}

// makeRemoteSyncName constructs the name of the RemoteSync object
// by prefixing rolloutName with clusterName.
// For example, RemoteSync object's name for a rollout `app-rollout` and
// target cluster `gke-1` will be `gke-1-app-rollout`.
func makeRemoteSyncName(clusterName, rolloutName string) string {
	return fmt.Sprintf("%s-%s", clusterName, rolloutName)
}

// getExternalSyncName returns the name of the RSync object's name.
// It is derived by stripping away the cluster-name prefix
// from the RemoteSync object's name. We use rollout-name as RSync object's name.
func getExternalSyncName(rrs *gitopsv1alpha1.RemoteSync) string {
	clusterRef := rrs.Spec.ClusterRef
	clusterName := clusterRef.Name[strings.LastIndex(clusterRef.Name, "/")+1:]
	return strings.TrimPrefix(rrs.GetName(), fmt.Sprintf("%s-", clusterName))
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.channel = make(chan event.GenericEvent, 10)
	r.watchers = make(map[gitopsv1alpha1.ClusterRef]*watcher)
	r.Client = mgr.GetClient()
	gkeclusterapis.AddToScheme(mgr.GetScheme())

	if err := gitopsv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	clusterStore, err := clusterstore.NewClusterStore(r.Client, mgr.GetConfig())
	if err != nil {
		return err
	}
	r.store = clusterStore

	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1alpha1.RemoteSync{}).
		Watches(
			&source.Channel{Source: r.channel},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				logger := klog.NewKlogr().WithValues("controller", "remotesync")

				var rsName string
				var rsNamespace string
				if o.GetLabels() != nil {
					rsName = o.GetLabels()[remoteSyncNameLabel]
					rsNamespace = o.GetLabels()[remoteSyncNamespaceLabel]
				}
				if rsName == "" || rsNamespace == "" {
					return []reconcile.Request{}
				}
				logger.Info("Resource contains a RemoteSync label", "resource", klog.KRef(o.GetNamespace(), o.GetName()), "remoteSync", klog.KRef(rsNamespace, rsName))
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Namespace: rsNamespace,
							Name:      rsName,
						},
					},
				}
			}),
		).
		Complete(r)
}
