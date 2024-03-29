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

package workloadidentitybinding

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/applyset"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/workloadidentitybindings/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName = "config.porch.kpt.dev/workloadidentitybindings"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

// WorkloadIdentityBindingReconciler reconciles WorkloadIdentityBinding objects
type WorkloadIdentityBindingReconciler struct {
	Options

	client.Client

	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-workloadidentitybinding webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=workloadidentitybindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=workloadidentitybindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=workloadidentitybindings/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups=iam.cnrm.cloud.google.com,resources=iampolicymembers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iam.cnrm.cloud.google.com,resources=iamserviceaccounts,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;patch

// Reconcile implements the main kubernetes reconciliation loop.
func (r *WorkloadIdentityBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var subject api.WorkloadIdentityBinding
	if err := r.Get(ctx, req.NamespacedName, &subject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if subject.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&subject, finalizerName) {
			controllerutil.AddFinalizer(&subject, finalizerName)
			if err := r.Update(ctx, &subject); err != nil {
				klog.Warningf("failed to update %s after adding finalizer: %v", req.Name, err)
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&subject, finalizerName) {
			// // our finalizer is present, so lets remove the annotation from the SA
			if err := r.removeWiAnnotation(ctx, &subject); err != nil {
				// failed to remove the annotation, so return the error so it can be retried.
				klog.Warningf("failed to remove SA annotation from %s: %v", req.Name, err)
				return ctrl.Result{}, err
			}
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&subject, finalizerName)
			if err := r.Update(ctx, &subject); err != nil {
				klog.Warningf("failed to update %s after removing finalizer: %v", req.Name, err)
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var result ctrl.Result

	projectID, err := r.findProjectID(ctx, &subject)
	if err != nil {
		return result, err
	}

	results, err := r.applyToClusterRef(ctx, projectID, &subject)
	if err == nil && results.AllApplied() && results.AllHealthy() {
		// If the IAMPolicyMember has been installed and reconciled, we add the annotation.
		err = r.addWiAnnotation(ctx, projectID, &subject)
	}
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
	if err != nil {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Error"})
	} else if !results.AllApplied() {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "ApplyInProgress"})
	} else if !results.AllHealthy() {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "NotHealthy"})
	} else {
		meta.SetStatusCondition(conditions, metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Ready"})
	}

	// TODO: Check apply results and think about status conditions

	return true
}

func (r *WorkloadIdentityBindingReconciler) findProjectID(ctx context.Context, subject *api.WorkloadIdentityBinding) (string, error) {
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: subject.GetNamespace()}, ns); err != nil {
		return "", fmt.Errorf("error getting namespace %q: %w", subject.GetNamespace(), err)
	}

	parentProjectID := ns.GetAnnotations()["cnrm.cloud.google.com/project-id"]
	if parentProjectID == "" {
		return "", fmt.Errorf("project-id not found for namespace %q", ns.GetName())
	}
	return parentProjectID, nil
}

func (r *WorkloadIdentityBindingReconciler) applyToClusterRef(ctx context.Context, projectID string, subject *api.WorkloadIdentityBinding) (*applyset.ApplyResults, error) {
	objects, err := r.BuildObjectsToApply(ctx, projectID, subject)
	if err != nil {
		return nil, err
	}

	// TODO: Cache applyset?
	patchOptions := metav1.PatchOptions{
		FieldManager: fieldManager(subject),
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

func (r *WorkloadIdentityBindingReconciler) BuildObjectsToApply(ctx context.Context, projectID string, subject *api.WorkloadIdentityBinding) ([]applyset.ApplyableObject, error) {
	var objects []applyset.ApplyableObject

	{
		u := &unstructured.Unstructured{}
		u.SetAPIVersion("iam.cnrm.cloud.google.com/v1beta1")
		u.SetKind("IAMPolicyMember")
		u.SetName("workloadidentitybinding-" + subject.GetName())
		u.SetNamespace(subject.GetNamespace())

		saRef := subject.Spec.KubernetesServiceAccountRef

		saNamespace := saRef.Namespace
		saName := saRef.Name
		if saNamespace == "" {
			saNamespace = subject.GetNamespace()
		}

		member := "serviceAccount:" + projectID + ".svc.id.goog[" + saNamespace + "/" + saName + "]"

		resourceRef := map[string]string{
			"apiVersion": "iam.cnrm.cloud.google.com/v1beta1",
			"kind":       "IAMServiceAccount",
			"name":       subject.Spec.GcpServiceAccountRef.Name,
		}
		if ns := subject.Spec.GcpServiceAccountRef.Namespace; ns != "" {
			resourceRef["namespace"] = ns
		}
		if ext := subject.Spec.GcpServiceAccountRef.External; ext != "" {
			resourceRef["external"] = ext
		}
		u.Object["spec"] = map[string]interface{}{
			"member":      member,
			"role":        "roles/iam.workloadIdentityUser",
			"resourceRef": resourceRef,
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

func (r *WorkloadIdentityBindingReconciler) addWiAnnotation(ctx context.Context, projectID string, subject *api.WorkloadIdentityBinding) error {
	// TODO: We should have a watch here so we can annotate the ServiceAccount even if it is
	// applied later.
	ksa, err := r.getKubernetesServiceAccount(ctx, subject)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// TODO: Same as above, we should have a watch here so we can annotate the ksa when
	// the gsa is created and reconciled (i.e. have the status.email field set)
	gsa, err := r.getGcpServiceAccount(ctx, subject)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	gsaEmail, found, err := unstructured.NestedString(gsa.Object, "status", "email")
	if err != nil {
		return err
	}
	if !found || gsaEmail == "" {
		return nil
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
	u.SetName(ksa.GetName())
	u.SetNamespace(ksa.GetNamespace())
	u.SetAnnotations(map[string]string{
		"iam.gke.io/gcp-service-account": gsaEmail,
	})

	return r.updateServiceAccount(ctx, u, subject)
}

func (r *WorkloadIdentityBindingReconciler) removeWiAnnotation(ctx context.Context, subject *api.WorkloadIdentityBinding) error {
	sa, err := r.getKubernetesServiceAccount(ctx, subject)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
	u.SetName(sa.GetName())
	u.SetNamespace(sa.GetNamespace())
	u.SetAnnotations(make(map[string]string))

	return r.updateServiceAccount(ctx, u, subject)
}

func (r *WorkloadIdentityBindingReconciler) getKubernetesServiceAccount(ctx context.Context, subject *api.WorkloadIdentityBinding) (*unstructured.Unstructured, error) {
	saRef := subject.Spec.KubernetesServiceAccountRef

	saName := saRef.Name
	saNamespace := saRef.Namespace
	if saNamespace == "" {
		saNamespace = subject.GetNamespace()
	}

	var sa unstructured.Unstructured
	sa.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
	if err := r.Get(ctx, types.NamespacedName{Name: saName, Namespace: saNamespace}, &sa); err != nil {
		return nil, err
	}

	return &sa, nil
}

func (r *WorkloadIdentityBindingReconciler) getGcpServiceAccount(ctx context.Context, subject *api.WorkloadIdentityBinding) (*unstructured.Unstructured, error) {
	gsaRef := subject.Spec.GcpServiceAccountRef

	gsaName := gsaRef.Name
	gsaNamespace := gsaRef.Namespace
	if gsaNamespace == "" {
		gsaNamespace = subject.GetNamespace()
	}

	u := &unstructured.Unstructured{}
	u.SetAPIVersion("iam.cnrm.cloud.google.com/v1beta1")
	u.SetKind("IAMServiceAccount")

	err := r.Get(ctx, types.NamespacedName{Name: gsaName, Namespace: gsaNamespace}, u)
	return u, err
}

func (r *WorkloadIdentityBindingReconciler) updateServiceAccount(ctx context.Context, sa *unstructured.Unstructured, subject *api.WorkloadIdentityBinding) error {
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}

	mapping, err := r.restMapper.RESTMapping(corev1.SchemeGroupVersion.WithKind("ServiceAccount").GroupKind(), corev1.SchemeGroupVersion.Version)
	if err != nil {
		return err
	}
	dr := r.dynamicClient.Resource(mapping.Resource)
	_, err = dr.Namespace(sa.GetNamespace()).Patch(ctx, sa.GetName(), types.ApplyPatchType, data, metav1.PatchOptions{
		FieldManager: fieldManager(subject),
	})
	return client.IgnoreNotFound(err)
}

func fieldManager(subject *api.WorkloadIdentityBinding) string {
	return subject.GetObjectKind().GroupVersionKind().Kind + "-" + subject.GetNamespace() + "-" + subject.GetName()
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
