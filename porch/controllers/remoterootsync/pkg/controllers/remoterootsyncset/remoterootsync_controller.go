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
	"os"
	"path"
	"strings"
	"unicode"

	kptoci "github.com/GoogleContainerTools/kpt/pkg/oci"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/pkg/applyset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/pkg/remoteclient"
	"github.com/GoogleContainerTools/kpt/porch/pkg/oci"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var (
	RootSyncNamespace  = "config-management-system"
	RootSyncApiVersion = "configsync.gke.io/v1beta1"
	RootSyncName       = "root-sync"
	RootSyncKind       = "RootSync"
)

// RemoteRootSyncSetReconciler reconciles RemoteRootSyncSet objects
type RemoteRootSyncSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	ociStorage *kptoci.Storage

	// localRESTConfig stores the local RESTConfig from the manager
	// This is currently (only) used in "development" mode, for loopback configuration
	localRESTConfig *rest.Config
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

	var result ctrl.Result

	var patchErrs []error
	for _, clusterRef := range subject.Spec.ClusterRefs {
		results, err := r.applyToClusterRef(ctx, &subject, clusterRef)
		if err != nil {
			patchErrs = append(patchErrs, err)
		}
		if updateTargetStatus(&subject, clusterRef, results, err) {
			if err := r.Status().Update(ctx, &subject); err != nil {
				patchErrs = append(patchErrs, err)
			}
		}

		if results != nil && !(results.AllApplied() && results.AllHealthy()) {
			result.Requeue = true
		}
	}

	if len(patchErrs) != 0 {
		for _, patchErr := range patchErrs {
			klog.Errorf("%v", patchErr)
		}
		return ctrl.Result{}, patchErrs[0]
	}
	return result, nil
}

func updateTargetStatus(subject *api.RemoteRootSyncSet, ref *api.ClusterRef, applyResults *applyset.ApplyResults, err error) bool {
	var found *api.TargetStatus
	for i := range subject.Status.Targets {
		target := &subject.Status.Targets[i]
		if target.Ref == *ref {
			found = target
			break
		}
	}
	if found == nil {
		subject.Status.Targets = append(subject.Status.Targets, api.TargetStatus{
			Ref: *ref,
		})
		found = &subject.Status.Targets[len(subject.Status.Targets)-1]
	}

	if err != nil {
		meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionFalse, Reason: "Error", Message: err.Error()})
		meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "UpdateInProgress"})
	} else {
		if applyResults == nil {
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionFalse, Reason: "UnknownStatus"})
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "UnknownStatus"})
		} else if !applyResults.AllApplied() {
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionFalse, Reason: "UpdateInProgress"})
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "UpdateInProgress"})
		} else if !applyResults.AllHealthy() {
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "Applied"})
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "WaitingForReady"})
		} else {
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "Applied"})
			meta.SetStatusCondition(&found.Conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready"})
		}
	}
	// TODO: SetStatusCondition should return an indiciation if anything has changes

	return updateAggregateStatus(subject)
}

func updateAggregateStatus(subject *api.RemoteRootSyncSet) bool {
	// TODO: Verify that all targets are accounted for

	applied := make(map[string]int32)
	ready := make(map[string]int32)

	targetCount := int32(0)
	for _, status := range subject.Status.Targets {
		targetCount++
		appliedCondition := meta.FindStatusCondition(status.Conditions, "Applied")
		if appliedCondition == nil {
			applied["UnknownStatus"]++
		} else {
			applied[appliedCondition.Reason]++
		}
		readyCondition := meta.FindStatusCondition(status.Conditions, "Ready")
		if appliedCondition == nil {
			ready["UnknownStatus"]++
		} else {
			ready[readyCondition.Reason]++
		}
	}

	conditions := &subject.Status.AggregatedStatus.Conditions
	if applied["UpdateInProgress"] > 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "UpdateInProgress"})
	} else if applied["Error"] > 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "Error"})
	} else if applied["Applied"] >= targetCount {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "Applied"})
	} else {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Applied", Status: metav1.ConditionTrue, Reason: "UnknownStatus"})
	}

	if ready["UpdateInProgress"] > 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "UpdateInProgress"})
	} else if ready["WaitingForReady"] > 0 {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "WaitingForReady"})
	} else if ready["Ready"] >= targetCount {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready"})
	} else {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "UnknownStatus"})
	}

	subject.Status.AggregatedStatus.Targets = targetCount
	subject.Status.AggregatedStatus.Applied = applied["Applied"]
	subject.Status.AggregatedStatus.Ready = ready["Ready"]

	return true
}

func (r *RemoteRootSyncSetReconciler) applyToClusterRef(ctx context.Context, subject *api.RemoteRootSyncSet, clusterRef *api.ClusterRef) (*applyset.ApplyResults, error) {
	var restConfig *rest.Config

	if os.Getenv("HACK_ENABLE_LOOPBACK") != "" {
		if clusterRef.Name == "loopback!" {
			restConfig = r.localRESTConfig
			klog.Warningf("HACK: using loopback! configuration")
		}
	}

	if restConfig == nil {
		rc, err := remoteclient.GetRemoteClient(ctx, r.Client, clusterRef, subject.Namespace)
		if err != nil {
			return nil, err
		}
		restConfig = rc
	}

	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new dynamic client: %w", err)
	}

	// TODO: Use a better discovery client
	discovery, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("error building discovery client: %w", err)
	}

	cached := memory.NewMemCacheClient(discovery)

	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cached)

	objects, err := r.BuildObjectsToApply(ctx, subject)
	if err != nil {
		return nil, err
	}

	// TODO: Cache applyset
	patchOptions := metav1.PatchOptions{
		FieldManager: "remoterootsync-" + subject.GetNamespace() + "-" + subject.GetName(),
	}

	// We force to overcome errors like: Apply failed with 1 conflict: conflict with "kubectl-client-side-apply" using apps/v1: .spec.template.spec.containers[name="porch-server"].image
	// TODO: How to handle this better
	force := true
	patchOptions.Force = &force

	applyset, err := applyset.New(applyset.Options{
		RESTMapper:   restMapper,
		Client:       client,
		PatchOptions: patchOptions,
	})
	if err != nil {
		return nil, err
	}

	if err := applyset.ReplaceAllObjects(objects); err != nil {
		return nil, err
	}

	results, err := applyset.ApplyOnce(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to apply to cluster %v: %w", clusterRef, err)
	}

	// TODO: Implement pruning

	return results, nil
}

// BuildObjectsToApply config root sync
func (r *RemoteRootSyncSetReconciler) BuildObjectsToApply(ctx context.Context, subject *api.RemoteRootSyncSet) ([]applyset.ApplyableObject, error) {
	repository := subject.GetSpec().GetTemplate().GetOCI().GetRepository()
	if repository == "" {
		return nil, fmt.Errorf("spec.template.oci.repository is not set")
	}
	imageName, err := kptoci.ParseImageTagName(repository)
	if err != nil {
		return nil, fmt.Errorf("unable to parse image %q: %w", repository, err)
	}
	klog.Infof("image name %s -> %#v", repository, *imageName)

	digest, err := oci.LookupImageTag(ctx, r.ociStorage, *imageName)
	if err != nil {
		return nil, err
	}

	resources, err := oci.LoadResources(ctx, r.ociStorage, digest)
	if err != nil {
		return nil, err
	}

	var objects []applyset.ApplyableObject

	for filePath, fileContents := range resources.Contents {
		ext := path.Ext(filePath)
		ext = strings.ToLower(ext)

		parse := false
		switch ext {
		case ".yaml", ".yml":
			parse = true

		default:
			klog.Warningf("ignoring non-yaml file %s", filePath)
		}

		if !parse {
			continue
		}
		// TODO: Use https://github.com/kubernetes-sigs/kustomize/blob/a5b61016bb40c30dd1b0a78290b28b2330a0383e/kyaml/kio/byteio_reader.go#L170 or similar?
		for _, s := range strings.Split(fileContents, "\n---\n") {
			if isWhitespace(s) {
				continue
			}

			o := &unstructured.Unstructured{}
			if err := yaml.Unmarshal([]byte(s), &o); err != nil {
				return nil, fmt.Errorf("error parsing yaml from %s: %w", filePath, err)
			}

			// TODO: sync with kpt logic; skip objects marked with the local-only annotation
			objects = append(objects, o)
		}
	}

	return objects, nil
}

func isWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteRootSyncSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&api.RemoteRootSyncSet{}).
		Complete(r); err != nil {
		return err
	}

	cacheDir := "./.cache"

	ociStorage, err := kptoci.NewStorage(cacheDir)
	if err != nil {
		return err
	}

	r.ociStorage = ociStorage

	r.localRESTConfig = mgr.GetConfig()

	return nil
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
	klog.Infof("external resource %s delete Done!", RootSyncName)
	return nil
}
