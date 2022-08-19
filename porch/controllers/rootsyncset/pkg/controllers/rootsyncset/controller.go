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

package rootsyncset

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncset/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/googleurl"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
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

// RootSyncSetReconciler reconciles a RootSyncSet object
type RootSyncSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	WorkloadIdentityHelper
}

//+kubebuilder:rbac:groups=config.cloud.google.com,resources=rootsyncsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.cloud.google.com,resources=rootsyncsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.cloud.google.com,resources=rootsyncsets/finalizers,verbs=update

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
	myFinalizerName := "config.cloud.google.com/finalizer"
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
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&rootsyncset, myFinalizerName)
			if err := r.Update(ctx, &rootsyncset); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	var patchErrs []error
	for _, clusterRef := range rootsyncset.Spec.ClusterRefs {
		clusterRefName := clusterRef.Kind + ":" + clusterRef.Name
		client, err := r.GetClient(ctx, clusterRef, rootsyncset.Namespace)
		if err != nil {
			patchErrs = append(patchErrs, err)
			continue
		}
		rootSyncRes, newRootSync, err := BuildObjectsToApply(&rootsyncset)
		if err != nil {
			patchErrs = append(patchErrs, err)
			continue
		}
		data, err := json.Marshal(newRootSync)
		if err != nil {
			patchErrs = append(patchErrs, fmt.Errorf("failed to encode root sync to JSON: %w", err))
			continue
		}
		rs, err := client.Resource(rootSyncRes).Namespace(rootSyncNamespace).Patch(ctx, rootSyncName, types.ApplyPatchType, data, metav1.PatchOptions{FieldManager: req.Name})
		if err != nil {
			patchErrs = append(patchErrs, fmt.Errorf("failed to patch RootSync %s in cluster %s: %w", rootSyncNamespace+"/"+rootSyncName, clusterRefName, err))
		} else {
			klog.Infof("Create/Update resource %s as %v", rootSyncName, rs)
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

//BuildObjectsToApply config root sync
func BuildObjectsToApply(rootsyncset *v1alpha1.RootSyncSet) (schema.GroupVersionResource, *unstructured.Unstructured, error) {
	gv, err := schema.ParseGroupVersion(rootSyncApiVersion)
	if err != nil {
		return schema.GroupVersionResource{}, nil, fmt.Errorf("failed to parse group version when building object: %w", err)
	}
	rootSyncRes := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: "rootsyncs"}
	newRootSync, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rootsyncset.Spec.Template)
	newRootSync["apiVersion"] = rootSyncApiVersion
	newRootSync["kind"] = rootSyncKind
	newRootSync["metadata"] = map[string]string{"name": rootSyncName,
		"namespace": rootSyncNamespace}
	fmt.Printf("rootsync looks like %v", newRootSync)
	if err != nil {
		return schema.GroupVersionResource{}, nil, fmt.Errorf("failed to convert to unstructured type: %w", err)
	}
	u := unstructured.Unstructured{Object: newRootSync}

	return rootSyncRes, &u, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RootSyncSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.WorkloadIdentityHelper.Init(mgr.GetConfig()); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RootSyncSet{}).
		Complete(r)
}

func (r *RootSyncSetReconciler) deleteExternalResources(ctx context.Context, rootsyncset *v1alpha1.RootSyncSet) error {
	var deleteErrs []error
	for _, clusterRef := range rootsyncset.Spec.ClusterRefs {
		myClient, err := r.GetClient(ctx, clusterRef, rootsyncset.Namespace)
		if err != nil {
			deleteErrs = append(deleteErrs, fmt.Errorf("failed to get client when delete resource: %w", err))
			continue
		}
		klog.Infof("deleting external resource %s ...", rootSyncName)
		gv, err := schema.ParseGroupVersion(rootSyncApiVersion)
		if err != nil {
			deleteErrs = append(deleteErrs, fmt.Errorf("failed to parse group version when deleting external resrouces: %w", err))
			continue
		}
		rootSyncRes := schema.GroupVersionResource{Group: gv.Group, Version: gv.Version, Resource: "rootsyncs"}
		err = myClient.Resource(rootSyncRes).Namespace("config-management-system").Delete(ctx, rootSyncName, metav1.DeleteOptions{})
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
	klog.Infof("external resource %s delete Done!", rootSyncName)
	return nil
}

// GetCCRESTConfig builds a rest.Config for accessing the config controller cluster,
// this is a tmp workaround.
func (r *RootSyncSetReconciler) GetCCRESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	gkeResourceLink, _, err := unstructured.NestedString(cluster.Object, "status", "gkeResourceLink")
	if err != nil {
		return nil, fmt.Errorf("failed to get status.gkeResourceLink field: %w", err)
	}
	if gkeResourceLink == "" {
		return nil, fmt.Errorf("status.gkeResourceLink not set in object")
	}

	googleURL, err := googleurl.ParseUnversioned(gkeResourceLink)
	if err != nil {
		return nil, fmt.Errorf("error parsing gkeResourceLink %q: %w", gkeResourceLink, err)
	}
	projectID := googleURL.Project
	location := googleURL.Location
	clusterName := googleURL.Extra["clusters"]

	tokenSource, err := r.GetConfigConnectorContextTokenSource(ctx, cluster.GetNamespace())
	if err != nil {
		return nil, err
	}

	c, err := container.NewClusterManagerClient(ctx, option.WithTokenSource(tokenSource), option.WithQuotaProject(projectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create new cluster manager client: %w", err)
	}
	defer c.Close()

	clusterSelfLink := "projects/" + projectID + "/locations/" + location + "/clusters/" + clusterName
	klog.Infof("cluster path is %s", clusterSelfLink)
	req := &containerpb.GetClusterRequest{
		Name: clusterSelfLink,
	}
	resp, err := c.GetCluster(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info for cluster %q: %w", clusterSelfLink, err)
	}
	restConfig := &rest.Config{}
	caData, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate from gke cluster %q: %w", clusterSelfLink, err)
	}
	restConfig.CAData = caData

	restConfig.Host = "https://" + resp.Endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}

	restConfig.BearerToken = token.AccessToken
	return restConfig, nil
}

func (r *RootSyncSetReconciler) GetClient(ctx context.Context, ref *v1alpha1.ClusterRef, ns string) (dynamic.Interface, error) {
	key := types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}
	if key.Namespace == "" {
		key.Namespace = ns
	}
	u := &unstructured.Unstructured{}
	var config *rest.Config
	gv, err := schema.ParseGroupVersion(ref.ApiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse group version when building object: %w", err)
	}

	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    ref.Kind,
	})
	if err := r.Get(ctx, key, u); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	if ref.Kind == "ContainerCluster" {
		config, err = r.GetGKERESTConfig(ctx, u)
	} else if ref.Kind == "ConfigControllerInstance" {
		config, err = r.GetCCRESTConfig(ctx, u) //TODO: tmp workaround, update after ACP add new fields
	} else {
		return nil, fmt.Errorf("failed to find target cluster, cluster kind has to be ContainerCluster or ConfigControllerInstance")
	}
	if err != nil {
		return nil, err
	}
	myClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new dynamic client: %w", err)
	}
	return myClient, nil
}

// GetGKERESTConfig builds a rest.Config for accessing the specified cluster,
// without assuming that kubeconfig is correctly configured / mapped.
func (r *RootSyncSetReconciler) GetGKERESTConfig(ctx context.Context, cluster *unstructured.Unstructured) (*rest.Config, error) {
	restConfig := &rest.Config{}
	clusterCaCertificate, exist, err := unstructured.NestedString(cluster.Object, "spec", "masterAuth", "clusterCaCertificate")
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("clusterCaCertificate field does not exist")
	}
	caData, err := base64.StdEncoding.DecodeString(clusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("error decoding ca certificate: %w", err)
	}
	restConfig.CAData = caData
	endpoint, exist, err := unstructured.NestedString(cluster.Object, "status", "endpoint")
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}
	if !exist {
		return nil, fmt.Errorf("endpoint field does not exist")
	}
	restConfig.Host = "https://" + endpoint
	klog.Infof("Host endpoint is %s", restConfig.Host)
	tokenSource, err := r.GetConfigConnectorContextTokenSource(ctx, cluster.GetNamespace())
	if err != nil {
		return nil, err
	}
	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("error getting token: %w", err)
	}
	restConfig.BearerToken = token.AccessToken
	return restConfig, nil
}

// GetConfigConnectorContextTokenSource gets and returns the ConfigConnectorContext for the given namespace.
func (r *RootSyncSetReconciler) GetConfigConnectorContextTokenSource(ctx context.Context, ns string) (oauth2.TokenSource, error) {
	gvr := schema.GroupVersionResource{
		Group:    "core.cnrm.cloud.google.com",
		Version:  "v1beta1",
		Resource: "configconnectorcontexts",
	}

	cr, err := r.dynamicClient.Resource(gvr).Namespace(ns).Get(ctx, "configconnectorcontext.core.cnrm.cloud.google.com", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	googleServiceAccount, _, err := unstructured.NestedString(cr.Object, "spec", "googleServiceAccount")
	if err != nil {
		return nil, fmt.Errorf("error reading spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	if googleServiceAccount == "" {
		return nil, fmt.Errorf("could not find spec.googleServiceAccount from ConfigConnectorContext in %q: %w", ns, err)
	}

	kubeServiceAccount := types.NamespacedName{
		Namespace: "cnrm-system",
		Name:      "cnrm-controller-manager-" + ns,
	}
	return r.WorkloadIdentityHelper.GetGcloudAccessTokenSource(ctx, kubeServiceAccount, googleServiceAccount)
}
