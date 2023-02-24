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
	rootSyncNamespace = "config-management-system"
	rootSyncGVK       = schema.GroupVersionKind{
		Group:   "configsync.gke.io",
		Version: "v1beta1",
		Kind:    "RootSync",
	}
	rootSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "rootsyncs",
	}

	remoteRootSyncNameLabel      = "gitops.kpt.dev/remoterootsync-name"
	remoteRootSyncNamespaceLabel = "gitops.kpt.dev/remoterootsync-namespace"
)

const (
	conditionReady       = "Ready"
	conditionReconciling = "Reconciling"
	conditionStalled     = "Stalled"

	reasonSyncNotCreated       = "SyncNotCreated"
	reasonPendingReconcilation = "PendingReconcilation"
	reasonReady                = "Ready"
	reasonReconciling          = "Reconciling"
	reasonError                = "Error"
)

// RemoteRootSyncReconciler reconciles a RemoteRootSync object
type RemoteRootSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	store *clusterstore.ClusterStore

	// channel is where watchers put events to trigger new reconcilations based
	// on watch events from target clusters.
	channel chan event.GenericEvent

	mutex sync.Mutex

	watchers map[gitopsv1alpha1.ClusterRef]*watcher
}

//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remoterootsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remoterootsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=remoterootsyncs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RemoteRootSync object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *RemoteRootSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := klog.NewKlogr().WithValues("controller", "remoterootsync", "remoteRootSync", req.NamespacedName)
	ctx = klog.NewContext(ctx, logger)

	logger.Info("Reconciling")

	var remoterootsync gitopsv1alpha1.RemoteRootSync
	if err := r.Get(ctx, req.NamespacedName, &remoterootsync); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	myFinalizerName := "remoterootsync.gitops.kpt.dev/finalizer"
	if remoterootsync.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&remoterootsync, myFinalizerName) {
			controllerutil.AddFinalizer(&remoterootsync, myFinalizerName)
			if err := r.Update(ctx, &remoterootsync); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&remoterootsync, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if r.isExternalSyncCreated(&remoterootsync) {
				// Delete the external sync resource
				err := r.deleteExternalResources(ctx, &remoterootsync)
				if err != nil && !apierrors.IsNotFound(err) {
					statusError := r.updateStatus(ctx, &remoterootsync, "", err)

					if statusError != nil {
						logger.Error(statusError, "Failed to update status")
					}

					// if fail to delete the external dependency here, return with error
					// so that it can be retried
					return ctrl.Result{}, fmt.Errorf("have problem to delete external resource: %w", err)
				}

				// Make sure we stop any watches that are no longer needed.
				logger.Info("Pruning watches")
				r.pruneWatches(req.NamespacedName, &remoterootsync.Spec.ClusterRef)
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&remoterootsync, myFinalizerName)
			if err := r.Update(ctx, &remoterootsync); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	syncStatus, syncError := r.syncExternalSync(ctx, &remoterootsync)

	if err := r.updateStatus(ctx, &remoterootsync, syncStatus, syncError); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, syncError
}

func (r *RemoteRootSyncReconciler) syncExternalSync(ctx context.Context, rrs *gitopsv1alpha1.RemoteRootSync) (string, error) {
	syncName := rrs.Name
	clusterRef := &rrs.Spec.ClusterRef

	dynCl, err := r.getDynamicClientForCluster(ctx, clusterRef)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	if err := r.patchRootSync(ctx, dynCl, syncName, rrs); err != nil {
		return "", fmt.Errorf("failed to create/update sync: %w", err)
	}

	r.setupWatches(ctx, rrs.Name, rrs.Namespace, rrs.Spec.ClusterRef)

	syncStatus, err := checkSyncStatus(ctx, dynCl, syncName)
	if err != nil {
		return "", fmt.Errorf("faild to check status: %w", err)
	}

	return syncStatus, nil
}

func (r *RemoteRootSyncReconciler) updateStatus(ctx context.Context, rrs *gitopsv1alpha1.RemoteRootSync, syncStatus string, syncError error) error {
	logger := klog.FromContext(ctx)

	rrsPrior := rrs.DeepCopy()
	conditions := &rrs.Status.Conditions

	if syncError == nil {
		rrs.Status.SyncStatus = syncStatus

		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionReady, Status: metav1.ConditionTrue, Reason: reasonReady})
		meta.RemoveStatusCondition(conditions, conditionReconciling)
		meta.RemoveStatusCondition(conditions, conditionStalled)
	} else {
		readyReason := reasonPendingReconcilation
		readyStatus := metav1.ConditionUnknown

		rrs.Status.SyncStatus = "Unknown"

		if r.isExternalSyncCreated(rrs) {
		} else {
			rrs.Status.SyncStatus = ""
			readyReason = reasonSyncNotCreated
			readyStatus = metav1.ConditionFalse
		}

		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionReady, Status: readyStatus, Reason: readyReason})
		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionReconciling, Status: metav1.ConditionTrue, Reason: reasonReconciling})
		meta.SetStatusCondition(conditions, metav1.Condition{Type: conditionStalled, Status: metav1.ConditionTrue, Reason: reasonError, Message: syncError.Error()})
	}

	rrs.Status.ObservedGeneration = rrs.Generation

	if reflect.DeepEqual(rrs.Status, rrsPrior.Status) {
		return nil
	}

	logger.Info("Updating status")
	return r.Client.Status().Update(ctx, rrs)
}

// patchRootSync patches the RootSync in the remote clusters targeted by
// the clusterRefs based on the latest revision of the template in the RemoteRootSync.
func (r *RemoteRootSyncReconciler) patchRootSync(ctx context.Context, client dynamic.Interface, name string, rrs *gitopsv1alpha1.RemoteRootSync) error {
	logger := klog.FromContext(ctx)

	newRootSync, err := BuildObjectsToApply(rrs)
	if err != nil {
		return err
	}
	data, err := json.Marshal(newRootSync)
	if err != nil {
		return fmt.Errorf("failed to encode root sync to JSON: %w", err)
	}
	_, err = client.Resource(rootSyncGVR).Namespace(rootSyncNamespace).Patch(ctx, name, types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: name})
	if err != nil {
		return fmt.Errorf("failed to patch RootSync: %w", err)
	}

	logger.Info("RootSync resource created/updated", "rootSync", klog.KRef(rootSyncNamespace, name))
	return nil
}

// setupWatches makes sure we have the necessary watches running against
// the remote clusters we care about.
func (r *RemoteRootSyncReconciler) setupWatches(ctx context.Context, rrsName, ns string, clusterRef gitopsv1alpha1.ClusterRef) {
	logger := klog.FromContext(ctx)

	r.mutex.Lock()
	defer r.mutex.Unlock()
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      rrsName,
	}

	// If we already have a watch running, make sure we have the current RootSyncSet
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

// pruneWatches removes the current RootSyncSet from the liens map of all watchers
// that it no longer needs. If any of the watchers are no longer used by any RootSyncSets,
// they are shut down.
func (r *RemoteRootSyncReconciler) pruneWatches(rrsnn types.NamespacedName, clusterRef *gitopsv1alpha1.ClusterRef) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Look through all watchers to check if it used to be needed by the RootSyncSet
	// but is no longer.
	w, found := r.watchers[*clusterRef]
	if !found {
		return
	}

	// Delete the current RootSyncSet from the list of liens (it it exists)
	delete(w.liens, rrsnn)
	// If no other RootSyncSets need the watch, stop it and remove the watcher from the map.
	if len(w.liens) == 0 {
		w.cancelFunc()
		delete(r.watchers, *clusterRef)
	}
}

// BuildObjectsToApply config root sync
func BuildObjectsToApply(remoterootsync *gitopsv1alpha1.RemoteRootSync) (*unstructured.Unstructured, error) {
	newRootSync, err := runtime.DefaultUnstructuredConverter.ToUnstructured(remoterootsync.Spec.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to unstructured type: %w", err)
	}

	u := unstructured.Unstructured{Object: newRootSync}
	u.SetGroupVersionKind(rootSyncGVK)
	u.SetName(remoterootsync.Name)
	u.SetNamespace(rootSyncNamespace)

	labels := u.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[remoteRootSyncNameLabel] = remoterootsync.Name
	labels[remoteRootSyncNamespaceLabel] = remoterootsync.Namespace
	u.SetLabels(labels)

	return &u, nil
}

func (r *RemoteRootSyncReconciler) deleteExternalResources(ctx context.Context, remoterootsync *gitopsv1alpha1.RemoteRootSync) error {
	logger := klog.FromContext(ctx)

	clusterRef := &remoterootsync.Spec.ClusterRef
	dynCl, err := r.getDynamicClientForCluster(ctx, clusterRef)
	if err != nil {
		return err
	}

	logger.Info("Deleting external resource")
	err = dynCl.Resource(rootSyncGVR).Namespace("config-management-system").Delete(ctx, remoterootsync.Name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	logger.Info("External resource deleted")
	return err
}

func (r *RemoteRootSyncReconciler) getDynamicClientForCluster(ctx context.Context, clusterRef *gitopsv1alpha1.ClusterRef) (dynamic.Interface, error) {
	restConfig, err := r.store.GetRESTConfig(ctx, clusterRef.Name)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return dynamicClient, nil
}

func (r *RemoteRootSyncReconciler) isExternalSyncCreated(rrs *gitopsv1alpha1.RemoteRootSync) bool {
	readyCondition := meta.FindStatusCondition(rrs.Status.Conditions, conditionReady)

	if readyCondition == nil || (readyCondition.Status != metav1.ConditionTrue && readyCondition.Reason == reasonSyncNotCreated) {
		return false
	}

	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteRootSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
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
		For(&gitopsv1alpha1.RemoteRootSync{}).
		Watches(
			&source.Channel{Source: r.channel},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				logger := klog.NewKlogr().WithValues("controller", "remoterootsync")

				var rrsName string
				var rrsNamespace string
				if o.GetLabels() != nil {
					rrsName = o.GetLabels()[remoteRootSyncNameLabel]
					rrsNamespace = o.GetLabels()[remoteRootSyncNamespaceLabel]
				}
				if rrsName == "" || rrsNamespace == "" {
					return []reconcile.Request{}
				}
				logger.Info("Resource contains a RemoteRootSync label", "resource", klog.KRef(o.GetNamespace(), o.GetName()), "remoteRootSync", klog.KRef(rrsNamespace, rrsName))
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Namespace: rrsNamespace,
							Name:      rrsName,
						},
					},
				}
			}),
		).
		Complete(r)
}
