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

package remoterootsyncset

import (
	"context"
	"fmt"

	api "github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/pkg/remoteclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	rootSyncNamespace  = "config-management-system"
	rootSyncApiVersion = "configsync.gke.io/v1beta1"
	rootSyncName       = "root-sync"
	rootSyncKind       = "RootSync"
)

// RemoteRootSyncSetReconciler reconciles RemoteRootSyncSet objects
type RemoteRootSyncSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=config.cloud.google.com,resources=remoterootsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.cloud.google.com,resources=remoterootsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.cloud.google.com,resources=remoterootsyncs/finalizers,verbs=update

// Reconcile implements the main kubernetes reconciliation loop.
func (r *RemoteRootSyncSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var subject api.RemoteRootSyncSet
	if err := r.Get(ctx, req.NamespacedName, &subject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	myFinalizerName := "config.cloud.google.com/finalizer"
	if subject.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&subject, myFinalizerName) {
			controllerutil.AddFinalizer(&subject, myFinalizerName)
			if err := r.Update(ctx, &subject); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&subject, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, &subject); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, fmt.Errorf("have problem to delete external resource: %w", err)
			}
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&subject, myFinalizerName)
			if err := r.Update(ctx, &subject); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	var patchErrs []error
	for _, clusterRef := range subject.Spec.ClusterRefs {
		if err := r.applyToClusterRef(ctx, &subject, clusterRef); err != nil {
			patchErrs = append(patchErrs, err)
			continue
		}

	}
	if len(patchErrs) != 0 {
		for _, patchErr := range patchErrs {
			klog.Errorf("%v", patchErr)
		}
		return ctrl.Result{}, patchErrs[0]
	}
	return ctrl.Result{}, nil
}

func (r *RemoteRootSyncSetReconciler) applyToClusterRef(ctx context.Context, subject *api.RemoteRootSyncSet, clusterRef *api.ClusterRef) error {
	restConfig, err := remoteclient.GetRemoteClient(ctx, r.Client, clusterRef, subject.Namespace)
	if err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create a new dynamic client: %w", err)
	}

	// TODO: Use a better discovery client
	discovery, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("error building discovery client: %w", err)
	}

	cached := memory.NewMemCacheClient(discovery)

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cached)

	objects, err := BuildObjectsToApply(subject)
	if err != nil {
		return err
	}

	patchOptions := metav1.PatchOptions{FieldManager: "remoterootsync-" + subject.GetNamespace() + "-" + subject.GetName()}
	if err := applyObjects(ctx, restMapper, client, objects, patchOptions); err != nil {
		return fmt.Errorf("failed to apply to cluster %v: %w", clusterRef, err)
	}

	return nil
}

// BuildObjectsToApply config root sync
func BuildObjectsToApply(subject *api.RemoteRootSyncSet) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	ns := &unstructured.Unstructured{}
	ns.SetName("foo")
	ns.SetAPIVersion("v1")
	ns.SetKind("Namespace")
	ns.SetAnnotations(map[string]string{
		"created-by": subject.GetNamespace() + "-" + subject.GetName(),
	})

	objects = append(objects, ns)

	return objects, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteRootSyncSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.RemoteRootSyncSet{}).
		Complete(r)
}

func (r *RemoteRootSyncSetReconciler) deleteExternalResources(ctx context.Context, rootsyncset *api.RemoteRootSyncSet) error {
	var deleteErrs []error
	// for _, clusterRef := range rootsyncset.Spec.ClusterRefs {
	// 	myClient, err := remoteclient.GetRemoteClient(ctx, r.Client, clusterRef, rootsyncset.Namespace)
	// 	if err != nil {
	// 		deleteErrs = append(deleteErrs, fmt.Errorf("failed to get client when delete resource: %w", err))
	// 		continue
	// 	}
	// 	klog.Infof("deleting external resource %s ...", rootSyncName)
	// 	gv, err := schema.ParseGroupVersion(rootSyncApiVersion)
	// 	if err != nil {
	// 		deleteErrs = append(deleteErrs, fmt.Errorf("failed to parse group version when deleting external resrouces: %w", err))
	// 		continue
	// 	}
	// 	rootSyncRes := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: "rootsyncs"}
	// 	err = myClient.Resource(rootSyncRes).Namespace("config-management-system").Delete(ctx, rootSyncName, metav1.DeleteOptions{})
	// 	if err != nil && !apierrors.IsNotFound(err) {
	// 		deleteErrs = append(deleteErrs, fmt.Errorf("failed to delete external resource : %w", err))
	// 	}
	// }
	if len(deleteErrs) != 0 {
		for _, deleteErr := range deleteErrs {
			klog.Errorf("%v", deleteErr)
		}
		return deleteErrs[0]
	}
	klog.Infof("external resource %s delete Done!", rootSyncName)
	return nil
}
