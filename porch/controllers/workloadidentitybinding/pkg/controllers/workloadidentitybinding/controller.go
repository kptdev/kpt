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

package workloadidentitybinding

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/pkg/applyset"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/workloadidentitybinding/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkloadIdentityBindingReconciler reconciles WorkloadIdentityBinding objects
type WorkloadIdentityBindingReconciler struct {
	client.Client
	// Scheme *runtime.Scheme

	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

//+kubebuilder:rbac:groups=porch.kpt.dev,resources=workloadidentitybindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=workloadidentitybindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=workloadidentitybindings/finalizers,verbs=update

// Reconcile implements the main kubernetes reconciliation loop.
func (r *WorkloadIdentityBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var subject api.WorkloadIdentityBinding
	if err := r.Get(ctx, req.NamespacedName, &subject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// myFinalizerName := "config.cloud.google.com/finalizer"
	// if subject.ObjectMeta.DeletionTimestamp.IsZero() {
	// 	// The object is not being deleted, so if it does not have our finalizer,
	// 	// then lets add the finalizer and update the object. This is equivalent
	// 	// registering our finalizer.
	// 	if !controllerutil.ContainsFinalizer(&subject, myFinalizerName) {
	// 		controllerutil.AddFinalizer(&subject, myFinalizerName)
	// 		if err := r.Update(ctx, &subject); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}
	// } else {
	// 	// The object is being deleted
	// 	if controllerutil.ContainsFinalizer(&subject, myFinalizerName) {
	// 		// // our finalizer is present, so lets handle any external dependency
	// 		// if err := r.deleteExternalResources(ctx, &subject); err != nil {
	// 		// 	// if fail to delete the external dependency here, return with error
	// 		// 	// so that it can be retried
	// 		// 	return ctrl.Result{}, fmt.Errorf("have problem to delete external resource: %w", err)
	// 		// }
	// 		// remove our finalizer from the list and update it.
	// 		controllerutil.RemoveFinalizer(&subject, myFinalizerName)
	// 		if err := r.Update(ctx, &subject); err != nil {
	// 			return ctrl.Result{}, fmt.Errorf("failed to update %s after delete finalizer: %w", req.Name, err)
	// 		}
	// 	}
	// 	// Stop reconciliation as the item is being deleted
	// 	return ctrl.Result{}, nil
	// }

	var result ctrl.Result

	results, err := r.applyToClusterRef(ctx, &subject)
	if updateStatus(&subject, results, err) {
		if updateErr := r.Status().Update(ctx, &subject); updateErr != nil {
			if err == nil {
				return result, updateErr
			}
		}
	}

	if err != nil {
		klog.Warningf("error during apply: %v", err)
		// TODO: Post event
		return result, err
	}
	if results != nil && !(results.AllApplied() && results.AllHealthy()) {
		result.Requeue = true
	}

	return result, nil
}

func updateStatus(subject *api.WorkloadIdentityBinding, results *applyset.ApplyResults, err error) bool {
	conditions := &subject.Status.Conditions
	if err == nil {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready"})
	} else {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Error"})
	}

	// TODO: Check apply results and think about status conditions

	return true
}

func (r *WorkloadIdentityBindingReconciler) applyToClusterRef(ctx context.Context, subject *api.WorkloadIdentityBinding) (*applyset.ApplyResults, error) {

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: subject.GetNamespace()}, ns); err != nil {
		return nil, fmt.Errorf("error getting namespace %q: %w", subject.GetNamespace(), err)
	}

	objects, err := r.BuildObjectsToApply(ctx, ns, subject)
	if err != nil {
		return nil, err
	}

	// TODO: Cache applyset?
	patchOptions := metav1.PatchOptions{
		FieldManager: subject.GetObjectKind().GroupVersionKind().Kind + "-" + subject.GetNamespace() + "-" + subject.GetName(),
	}

	// We force to overcome errors like: Apply failed with 1 conflict: conflict with "kubectl-client-side-apply" using apps/v1: .spec.template.spec.containers[name="porch-server"].image
	// TODO: How to handle this better
	force := true
	patchOptions.Force = &force

	applyset, err := applyset.New(applyset.Options{
		RESTMapper:   r.restMapper,
		Client:       r.dynamicClient,
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
		return nil, fmt.Errorf("failed to apply: %w", err)
	}

	return results, nil
}

func (r *WorkloadIdentityBindingReconciler) BuildObjectsToApply(ctx context.Context, ns *corev1.Namespace, subject *api.WorkloadIdentityBinding) ([]applyset.ApplyableObject, error) {
	var objects []applyset.ApplyableObject

	{
		u := &unstructured.Unstructured{}
		u.SetAPIVersion("iam.cnrm.cloud.google.com/v1beta1")
		u.SetKind("IAMPolicyMember")
		u.SetName("workloadidentitybinding-" + subject.GetName())
		u.SetNamespace(subject.GetNamespace())

		saRef := subject.Spec.ServiceAccountRef

		saNamespace := saRef.Namespace
		saName := saRef.Name
		if saNamespace == "" {
			saNamespace = subject.GetNamespace()
		}

		parentProjectID := ns.GetAnnotations()["cnrm.cloud.google.com/project-id"]
		if parentProjectID == "" {
			return nil, fmt.Errorf("project-id not found for namespace %q", ns.GetName())
		}
		member := "serviceAccount:" + parentProjectID + ".svc.id.goog[" + saNamespace + "/" + saName + "]"

		u.Object["spec"] = map[string]interface{}{
			"member":      member,
			"role":        "roles/iam.workloadIdentityUser",
			"resourceRef": subject.Spec.ResourceRef,
		}

		objects = append(objects, u)
	}

	for _, obj := range objects {
		ownerRefs := obj.GetOwnerReferences()

		controller := true
		ownerRefs = append(ownerRefs, metav1.OwnerReference{
			APIVersion: subject.APIVersion,
			Kind:       subject.Kind,
			Name:       subject.GetName(),
			UID:        subject.GetUID(),
			Controller: &controller,
		})
		obj.SetOwnerReferences(ownerRefs)
	}
	return objects, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkloadIdentityBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&api.WorkloadIdentityBinding{}).
		Complete(r); err != nil {
		return err
	}

	restConfig := mgr.GetConfig()

	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create a new dynamic client: %w", err)
	}
	r.dynamicClient = client

	r.restMapper = mgr.GetRESTMapper()

	return nil
}
