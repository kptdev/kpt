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

package rootsyncset

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/remoteclient"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncsets/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	rootSyncSetNameLabel      = "config.porch.kpt.dev/rootsyncset-name"
	rootSyncSetNamespaceLabel = "config.porch.kpt.dev/rootsyncset-namespace"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

// RootSyncSetReconciler reconciles a RootSyncSet object
type RootSyncSetReconciler struct {
	Options

	remoteclient.RemoteClientGetter

	client.Client

	// channel is where watchers put events to trigger new reconcilations based
	// on watch events from target clusters.
	channel chan event.GenericEvent

	mutex    sync.Mutex
	watchers map[v1alpha1.ClusterRef]*watcher
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-rootsyncsets webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=rootsyncsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=configcontroller.cnrm.cloud.google.com,resources=configcontrollerinstances,verbs=get;list;watch
//+kubebuilder:rbac:groups=container.cnrm.cloud.google.com,resources=containerclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=core.cnrm.cloud.google.com,resources=configconnectorcontexts,verbs=get;list;watch
//+kubebuilder:rbac:groups=hub.gke.io,resources=memberships,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts/token,verbs=create

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Reconcile function compares the state specified by
// the RootSyncSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RootSyncSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var rootsyncset v1alpha1.RootSyncSet
	if err := r.Get(ctx, req.NamespacedName, &rootsyncset); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	myFinalizerName := "config.porch.kpt.dev/finalizer"
	if rootsyncset.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&rootsyncset, myFinalizerName) {
			controllerutil.AddFinalizer(&rootsyncset, myFinalizerName)
			if err := r.Update(ctx, &rootsyncset); err != nil {
				return ctrl.Result{}, fmt.Errorf("error adding finalizer: %w", err)
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&rootsyncset, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, &rootsyncset); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, fmt.Errorf("have problem to delete external resource: %w", err)
			}
			// Make sure we stop any watches that are no longer needed.
			r.pruneWatches(req.NamespacedName, []*v1alpha1.ClusterRef{})
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&rootsyncset, myFinalizerName)
			if err := r.Update(ctx, &rootsyncset); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	results := make(reconcileResult)
	for _, clusterRef := range rootsyncset.Spec.ClusterRefs {
		result := clusterRefResult{}
		clusterRefName := clusterRef.Kind + ":" + clusterRef.Name

		remoteClient, err := r.GetRemoteClient(ctx, clusterRef, rootsyncset.Namespace)
		if err != nil {
			result.clientError = err
			results[clusterRefName] = result
			continue
		}
		dynamicClient, err := remoteClient.DynamicClient()
		if err != nil {
			result.clientError = err
			results[clusterRefName] = result
			continue
		}
		r.setupWatches(ctx, dynamicClient, rootsyncset.Name, rootsyncset.Namespace, *clusterRef)

		if err := r.patchRootSync(ctx, dynamicClient, req.Name, &rootsyncset); err != nil {
			result.patchError = err
		}

		s, err := checkSyncStatus(ctx, dynamicClient, req.Name)
		if err != nil {
			result.statusError = err
			result.status = "Unknown"
		} else {
			result.status = s
		}

		results[clusterRefName] = result
	}

	r.pruneWatches(req.NamespacedName, rootsyncset.Spec.ClusterRefs)

	if err := r.updateStatus(ctx, &rootsyncset, results); err != nil {
		klog.Errorf("failed to update status: %w", err)
		return ctrl.Result{}, err
	}

	if errs := results.Errors(); len(errs) > 0 {
		klog.Warningf("Errors: %s", results.Error())
		return ctrl.Result{}, results
	}

	return ctrl.Result{}, nil
}

func (r *RootSyncSetReconciler) updateStatus(ctx context.Context, rss *v1alpha1.RootSyncSet, results reconcileResult) error {
	crss := make([]v1alpha1.ClusterRefStatus, 0)

	for _, clusterRef := range rss.Spec.ClusterRefs {
		clusterRefName := clusterRef.Kind + ":" + clusterRef.Name
		res := results[clusterRefName]
		crss = append(crss, v1alpha1.ClusterRefStatus{
			ApiVersion: clusterRef.ApiVersion,
			Kind:       clusterRef.Kind,
			Name:       clusterRef.Name,
			Namespace:  clusterRef.Namespace,
			SyncStatus: res.status,
		})
	}

	// Don't update if there are no changes.
	if equality.Semantic.DeepEqual(rss.Status.ClusterRefStatuses, crss) &&
		rss.Generation == rss.Status.ObservedGeneration {
		return nil
	}

	rss.Status.ClusterRefStatuses = crss
	rss.Status.ObservedGeneration = rss.Generation
	return r.Client.Status().Update(ctx, rss)
}

type reconcileResult map[string]clusterRefResult

func (r reconcileResult) Errors() []error {
	var errs []error
	for _, crr := range r {
		if crr.clientError != nil {
			errs = append(errs, crr.clientError)
		}
		if crr.patchError != nil {
			errs = append(errs, crr.patchError)
		}
		if crr.statusError != nil {
			errs = append(errs, crr.statusError)
		}
	}
	return errs
}

// TODO: Improve the formatting of the printed errors here.
func (r reconcileResult) Error() string {
	var sb strings.Builder
	for clusterRef, res := range r {
		if res.clientError != nil {
			sb.WriteString(fmt.Sprintf("failed to create client for %s: %v\n", clusterRef, res.clientError))
		}
		if res.patchError != nil {
			sb.WriteString(fmt.Sprintf("failed to patch %s: %v\n", clusterRef, res.patchError))
		}
		if res.statusError != nil {
			sb.WriteString(fmt.Sprintf("failed to check status for %s: %v\n", clusterRef, res.statusError))
		}
	}
	return sb.String()
}

type clusterRefResult struct {
	clientError error
	patchError  error
	statusError error
	status      string
}

// patchRootSync patches the RootSync in the remote clusters targeted by
// the clusterRefs based on the latest revision of the template in the RootSyncSet.
func (r *RootSyncSetReconciler) patchRootSync(ctx context.Context, client dynamic.Interface, name string, rss *v1alpha1.RootSyncSet) error {
	newRootSync, err := BuildObjectsToApply(rss)
	if err != nil {
		return err
	}
	data, err := json.Marshal(newRootSync)
	if err != nil {
		return fmt.Errorf("failed to encode root sync to JSON: %w", err)
	}
	rs, err := client.Resource(rootSyncGVR).Namespace(rootSyncNamespace).Patch(ctx, name, types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: name})
	if err != nil {
		return fmt.Errorf("failed to patch RootSync: %w", err)
	}
	klog.Infof("Create/Update resource %s as %v", name, rs)
	return nil
}

// setupWatches makes sure we have the necessary watches running against
// the remote clusters we care about.
func (r *RootSyncSetReconciler) setupWatches(ctx context.Context, client dynamic.Interface, rssName, ns string, clusterRef v1alpha1.ClusterRef) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	nn := types.NamespacedName{
		Namespace: ns,
		Name:      rssName,
	}

	// If we already have a watch running, make sure we have the current RootSyncSet
	// listed in the liens map.
	if w, found := r.watchers[clusterRef]; found {
		w.liens[nn] = struct{}{}
		return
	}

	// Since we don't currently have a watch running, create a new watcher
	// and add it to the map of watchers.
	watcherCtx, cancelFunc := context.WithCancel(context.Background())
	w := &watcher{
		clusterRef: clusterRef,
		ctx:        watcherCtx,
		cancelFunc: cancelFunc,
		client:     client,
		channel:    r.channel,
		liens: map[types.NamespacedName]struct{}{
			nn: {},
		},
	}
	klog.Infof("Creating watcher for %v", clusterRef)
	go w.watch()
	r.watchers[clusterRef] = w
}

// pruneWatches removes the current RootSyncSet from the liens map of all watchers
// that it no longer needs. If any of the watchers are no longer used by any RootSyncSets,
// they are shut down.
func (r *RootSyncSetReconciler) pruneWatches(rssnn types.NamespacedName, clusterRefs []*v1alpha1.ClusterRef) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	klog.Infof("Pruning watches for %s which has %v clusterRefs", rssnn.String(), clusterRefs)

	// Look through all watchers to check if it used to be needed by the RootSyncSet
	// but is no longer.
	for clusterRef, w := range r.watchers {
		// If the watcher is still needed, we don't need to do anything.
		var found bool
		for _, cr := range clusterRefs {
			if clusterRef == *cr {
				found = true
			}
		}
		if found {
			continue
		}

		// Delete the current RootSyncSet from the list of liens (it it exists)
		delete(w.liens, rssnn)
		// If no other RootSyncSets need the watch, stop it and remove the watcher from the map.
		if len(w.liens) == 0 {
			w.cancelFunc()
			delete(r.watchers, clusterRef)
		}
	}
}

// BuildObjectsToApply config root sync
func BuildObjectsToApply(rootsyncset *v1alpha1.RootSyncSet) (*unstructured.Unstructured, error) {
	newRootSync, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rootsyncset.Spec.Template)
	if err != nil {
		return nil, err
	}
	u := unstructured.Unstructured{Object: newRootSync}
	u.SetGroupVersionKind(rootSyncGVK)
	u.SetName(rootsyncset.Name)
	u.SetNamespace(rootSyncNamespace)
	u.SetLabels(map[string]string{
		rootSyncSetNameLabel:      rootsyncset.Name,
		rootSyncSetNamespaceLabel: rootsyncset.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert to unstructured type: %w", err)
	}
	return &u, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootSyncSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.RemoteClientGetter.Init(mgr); err != nil {
		return err
	}

	r.channel = make(chan event.GenericEvent, 10)
	r.watchers = make(map[v1alpha1.ClusterRef]*watcher)

	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RootSyncSet{}).
		Watches(
			&source.Channel{Source: r.channel},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				var rssName string
				var rssNamespace string
				if o.GetLabels() != nil {
					rssName = o.GetLabels()[rootSyncSetNameLabel]
					rssNamespace = o.GetLabels()[rootSyncSetNamespaceLabel]
				}
				if rssName == "" || rssNamespace == "" {
					return []reconcile.Request{}
				}
				klog.Infof("Resource %s contains a label for %s", o.GetName(), rssName)
				return []reconcile.Request{
					{
						NamespacedName: types.NamespacedName{
							Namespace: rssNamespace,
							Name:      rssName,
						},
					},
				}
			}),
		).
		Complete(r)
}

func (r *RootSyncSetReconciler) deleteExternalResources(ctx context.Context, rootsyncset *v1alpha1.RootSyncSet) error {
	var deleteErrs []error
	for _, clusterRef := range rootsyncset.Spec.ClusterRefs {
		remoteClient, err := r.GetRemoteClient(ctx, clusterRef, rootsyncset.Namespace)
		if err != nil {
			deleteErrs = append(deleteErrs, fmt.Errorf("failed to get client when deleting resource: %w", err))
			continue
		}
		dynamicClient, err := remoteClient.DynamicClient()
		if err != nil {
			deleteErrs = append(deleteErrs, fmt.Errorf("failed to get client when deleting resource: %w", err))
			continue
		}
		klog.Infof("deleting external resource %s ...", rootsyncset.Name)
		err = dynamicClient.Resource(rootSyncGVR).Namespace("config-management-system").Delete(ctx, rootsyncset.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			deleteErrs = append(deleteErrs, fmt.Errorf("failed to delete external resource : %w", err))
		}
	}
	if len(deleteErrs) != 0 {
		for _, deleteErr := range deleteErrs {
			klog.Errorf("%v", deleteErr)
		}
		return deleteErrs[0]
	}
	klog.Infof("external resource %s delete Done!", rootsyncset.Name)
	return nil
}
