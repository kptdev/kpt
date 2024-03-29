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

package remoterootsyncset

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"

	kptoci "github.com/GoogleContainerTools/kpt/pkg/oci"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/applyset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/remoteclient"
	"github.com/GoogleContainerTools/kpt/porch/pkg/objects"
	"github.com/GoogleContainerTools/kpt/porch/pkg/oci"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	RootSyncNamespace  = "config-management-system"
	RootSyncApiVersion = "configsync.gke.io/v1beta1"
	RootSyncName       = "root-sync"
	RootSyncKind       = "RootSync"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

// RemoteRootSyncSetReconciler reconciles RemoteRootSyncSet objects
type RemoteRootSyncSetReconciler struct {
	Options

	remoteClientGetter remoteclient.RemoteClientGetter

	client client.Client

	// uncachedClient queries the apiserver without using a watch cache.
	// This is useful for PackageRevisionResources, which are large
	// and would consume a lot of memory, and so we deliberately don't
	// support watching them.
	uncachedClient client.Client

	ociStorage *kptoci.Storage

	// localRESTConfig stores the local RESTConfig from the manager
	// This is currently (only) used in "development" mode, for loopback configuration
	localRESTConfig *rest.Config
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-remoterootsyncsets webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=remoterootsyncsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=remoterootsyncsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=remoterootsyncsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions;packagerevisionresources,verbs=get;list;watch

//+kubebuilder:rbac:groups=configcontroller.cnrm.cloud.google.com,resources=configcontrollerinstances,verbs=get;list;watch
//+kubebuilder:rbac:groups=container.cnrm.cloud.google.com,resources=containerclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=gkehub.cnrm.cloud.google.com,resources=gkehubmemberships,verbs=get;list;watch

//+kubebuilder:rbac:groups=core.cnrm.cloud.google.com,resources=configconnectors;configconnectorcontexts,verbs=get;list;watch

// Reconcile implements the main kubernetes reconciliation loop.
func (r *RemoteRootSyncSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var subject api.RemoteRootSyncSet
	if err := r.client.Get(ctx, req.NamespacedName, &subject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	myFinalizerName := "config.porch.kpt.dev/finalizer"
	if subject.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&subject, myFinalizerName) {
			controllerutil.AddFinalizer(&subject, myFinalizerName)
			if err := r.client.Update(ctx, &subject); err != nil {
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
			if err := r.client.Update(ctx, &subject); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var result ctrl.Result

	var applyErrors []error
	for _, clusterRef := range subject.Spec.ClusterRefs {
		results, err := r.applyToClusterRef(ctx, &subject, clusterRef)
		if err != nil {
			klog.Errorf("error applying to ref %v: %v", clusterRef, err)
			applyErrors = append(applyErrors, err)
		}

		updateTargetStatus(&subject, clusterRef, results, err)

		// TODO: Do we ever want to do a partial flush of results?  Should we exit the loop and re-reconcile?

		if results != nil && !(results.AllApplied() && results.AllHealthy()) {
			result.Requeue = true
		}
	}

	specTargets := make(map[api.ClusterRef]bool)
	for _, ref := range subject.Spec.ClusterRefs {
		specTargets[*ref] = true
	}

	// Remove any old target statuses
	var keepTargets []api.TargetStatus
	for i := range subject.Status.Targets {
		target := &subject.Status.Targets[i]
		if specTargets[target.Ref] {
			keepTargets = append(keepTargets, *target)
		}
	}
	subject.Status.Targets = keepTargets

	updateAggregateStatus(&subject)

	// TODO: Do this in a lazy way?
	if err := r.client.Status().Update(ctx, &subject); err != nil {
		return result, fmt.Errorf("error updating status: %w", err)
	}

	if len(applyErrors) != 0 {
		return result, applyErrors[0]
	}
	return result, nil
}

func updateTargetStatus(subject *api.RemoteRootSyncSet, ref *api.ClusterRef, applyResults *applyset.ApplyResults, err error) {
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
	remoteClient, err := r.remoteClientGetter.GetRemoteClient(ctx, clusterRef, subject.Namespace)
	if err != nil {
		return nil, err
	}

	restMapper, err := remoteClient.RESTMapper()
	if err != nil {
		return nil, err
	}

	dynamicClient, err := remoteClient.DynamicClient()
	if err != nil {
		return nil, err
	}

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
		Client:       dynamicClient,
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
	sourceFormat := subject.GetSpec().GetTemplate().GetSourceFormat()
	switch sourceFormat {
	case "oci":
		return r.buildObjectsToApplyFromOci(ctx, subject)
	case "package":
		return r.buildObjectsToApplyFromPackage(ctx, subject)
	default:
		return nil, fmt.Errorf("unknown sourceFormat %q", sourceFormat)
	}
}

func (r *RemoteRootSyncSetReconciler) buildObjectsToApplyFromOci(ctx context.Context, subject *api.RemoteRootSyncSet) ([]applyset.ApplyableObject, error) {
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

	unstructureds, err := objects.Parser{}.AsUnstructureds(resources.Contents)
	if err != nil {
		return nil, err
	}

	var applyables []applyset.ApplyableObject
	for _, u := range unstructureds {
		applyables = append(applyables, u)
	}
	return applyables, nil
}

func (r *RemoteRootSyncSetReconciler) buildObjectsToApplyFromPackage(ctx context.Context, subject *api.RemoteRootSyncSet) ([]applyset.ApplyableObject, error) {
	packageName := subject.GetSpec().GetTemplate().GetPackageRef().GetName()
	if packageName == "" {
		return nil, fmt.Errorf("spec.template.packageRef.name is not set")
	}

	ns := subject.GetNamespace()

	var packageRevisions porchapi.PackageRevisionList
	// Note that latest revision is planned for removal: #3672

	// TODO: publish package name as label?
	// TODO: Make package a first class concept?
	// TODO: Have some indicator of latest revision?
	if err := r.client.List(ctx, &packageRevisions, client.InNamespace(ns)); err != nil {
		// Not found here is unexpected
		return nil, fmt.Errorf("error listing package revisions: %w", err)
	}

	var latestPackageRevision *porchapi.PackageRevision
	for i := range packageRevisions.Items {
		candidate := &packageRevisions.Items[i]
		if candidate.Spec.PackageName != packageName {
			continue
		}
		if !strings.Contains(candidate.Spec.RepositoryName, "deployment") {
			// TODO: How can we only pick up deployment packages?  Probably labels...
			klog.Warningf("HACK: ignoring package that does not appear to be a deployment package")
			continue
		}

		candidateRevision := candidate.Spec.Revision
		if !strings.HasPrefix(candidateRevision, "v") {
			klog.Warningf("ignoring revision %q with unexpected format %q", candidate.Name, candidateRevision)
			continue
		}

		if latestPackageRevision == nil {
			latestPackageRevision = candidate
		} else {
			latestRevision := latestPackageRevision.Spec.Revision

			if !strings.HasPrefix(latestRevision, "v") {
				return nil, fmt.Errorf("unexpected revision format %q", latestRevision)
			}
			latestRevision = strings.TrimPrefix(latestRevision, "v")

			if !strings.HasPrefix(candidateRevision, "v") {
				return nil, fmt.Errorf("unexpected revision format %q", candidateRevision)
			}
			candidateRevision = strings.TrimPrefix(candidateRevision, "v")

			latestRevisionInt, err := strconv.Atoi(latestRevision)
			if err != nil {
				return nil, fmt.Errorf("unexpected revision format %q", latestRevision)
			}

			candidateRevisionInt, err := strconv.Atoi(candidateRevision)
			if err != nil {
				return nil, fmt.Errorf("unexpected revision format %q", candidateRevision)
			}

			if candidateRevisionInt == latestRevisionInt {
				return nil, fmt.Errorf("found two package revision with same revision: %q and %q", candidate.Name, latestPackageRevision.Name)
			}

			if candidateRevisionInt > latestRevisionInt {
				latestPackageRevision = candidate
			}
		}
	}
	if latestPackageRevision == nil {
		return nil, fmt.Errorf("cannot find latest version of package %q in namespace %q", packageName, ns)
	}

	id := types.NamespacedName{
		Namespace: latestPackageRevision.Namespace,
		Name:      latestPackageRevision.Name,
	}
	klog.Infof("found latest package %q", id)
	latestPackageRevisionResources := &porchapi.PackageRevisionResources{}
	if err := r.uncachedClient.Get(ctx, id, latestPackageRevisionResources); err != nil {
		// Not found here is unexpected
		return nil, fmt.Errorf("error getting package revision resources for %v: %w", id, err)
	}

	unstructureds, err := objects.Parser{}.AsUnstructureds(latestPackageRevisionResources.Spec.Resources)
	if err != nil {
		return nil, err
	}

	var applyables []applyset.ApplyableObject
	for _, u := range unstructureds {
		applyables = append(applyables, u)
	}
	return applyables, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RemoteRootSyncSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := porchapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if err := r.remoteClientGetter.Init(mgr); err != nil {
		return err
	}

	r.client = mgr.GetClient()

	// We need an uncachedClient to query objects directly.
	// In particular we don't want to watch PackageRevisionResources,
	// they are large so would have a large memory footprint,
	// and we don't want to support watch on them anyway.
	// If you need to watch PackageRevisionResources, you can watch PackageRevisions instead.
	uncachedClient, err := client.New(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
		Mapper: mgr.GetRESTMapper(),
	})
	if err != nil {
		return fmt.Errorf("creating uncached client: %w", err)
	}
	r.uncachedClient = uncachedClient

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
