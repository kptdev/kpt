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

package packagevariantset

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Options struct{}

func (o *Options) InitDefaults()                       {}
func (o *Options) BindFlags(_ string, _ *flag.FlagSet) {}

// PackageVariantSetReconciler reconciles a PackageVariantSet object
type PackageVariantSetReconciler struct {
	client.Client
	Options

	serializer *json.Serializer
}

const (
	PackageVariantSetOwnerLabel = "config.porch.kpt.dev/packagevariantset"

	ConditionTypeStalled = "Stalled" // whether or not the resource reconciliation is making progress or not
	ConditionTypeReady   = "Ready"   // whether or not the reconciliation succeeded

	PackageVariantNameMaxLength  = 63
	PackageVariantNameHashLength = 8
)

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-packagevariantsets webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets/finalizers,verbs=update
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariants,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups=*,resources=*,verbs=list

// Reconcile implements the main kubernetes reconciliation loop.
func (r *PackageVariantSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pvs, prList, repoList, err := r.init(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	if pvs == nil {
		// maybe the pvs was deleted
		return ctrl.Result{}, nil
	}

	defer func() {
		if err := r.Client.Status().Update(ctx, pvs); err != nil {
			klog.Errorf("could not update status: %w\n", err)
		}
	}()

	if errs := validatePackageVariantSet(pvs); len(errs) > 0 {
		setStalledConditionsToTrue(pvs, "ValidationError", combineErrors(errs))
		return ctrl.Result{}, nil
	}

	upstreamPR, err := r.getUpstreamPR(pvs.Spec.Upstream, prList)
	if err != nil {
		setStalledConditionsToTrue(pvs, "UpstreamNotFound", err.Error())
		// Currently we watch all PackageRevisions, so no need to requeue
		// here, as we will get triggered if a new upstream appears
		return ctrl.Result{}, nil
	}

	downstreams, err := r.unrollDownstreamTargets(ctx, pvs)
	if err != nil {
		if meta.IsNoMatchError(err) {
			setStalledConditionsToTrue(pvs, "NoMatchingTargets", err.Error())
			return ctrl.Result{}, nil
		}
		setStalledConditionsToTrue(pvs, "UnexpectedError", err.Error())
		return ctrl.Result{}, nil
	}

	meta.SetStatusCondition(&pvs.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeStalled,
		Status:  "False",
		Reason:  "Valid",
		Message: "all validation checks passed",
	})

	err = r.ensurePackageVariants(ctx, pvs, repoList, upstreamPR, downstreams)
	if err != nil {
		setStalledConditionsToTrue(pvs, "UnexpectedError", err.Error())
		return ctrl.Result{}, nil
	}

	meta.SetStatusCondition(&pvs.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  "True",
		Reason:  "Reconciled",
		Message: "package variants successfully reconciled",
	})

	return ctrl.Result{}, nil
}

func (r *PackageVariantSetReconciler) init(ctx context.Context, req ctrl.Request) (*api.PackageVariantSet,
	*porchapi.PackageRevisionList, *configapi.RepositoryList, error) {
	var pvs api.PackageVariantSet
	if err := r.Client.Get(ctx, req.NamespacedName, &pvs); err != nil {
		return nil, nil, nil, client.IgnoreNotFound(err)
	}

	var prList porchapi.PackageRevisionList
	if err := r.Client.List(ctx, &prList, client.InNamespace(pvs.Namespace)); err != nil {
		return nil, nil, nil, err
	}

	var repoList configapi.RepositoryList
	if err := r.Client.List(ctx, &repoList, client.InNamespace(pvs.Namespace)); err != nil {
		return nil, nil, nil, err
	}

	return &pvs, &prList, &repoList, nil
}

func (r *PackageVariantSetReconciler) getUpstreamPR(upstream *pkgvarapi.Upstream,
	prList *porchapi.PackageRevisionList) (*porchapi.PackageRevision, error) {

	for _, pr := range prList.Items {
		if pr.Spec.RepositoryName == upstream.Repo &&
			pr.Spec.PackageName == upstream.Package &&
			pr.Spec.Revision == upstream.Revision {
			return &pr, nil
		}
	}
	return nil, fmt.Errorf("could not find upstream package revision '%s/%s' in repo '%s'",
		upstream.Package, upstream.Revision, upstream.Repo)
}

type pvContext struct {
	template       *api.PackageVariantTemplate
	repoDefault    string
	packageDefault string
	object         *unstructured.Unstructured
}

func (r *PackageVariantSetReconciler) unrollDownstreamTargets(ctx context.Context,
	pvs *api.PackageVariantSet) ([]pvContext, error) {

	upstreamPackageName := pvs.Spec.Upstream.Package
	var result []pvContext

	for i, target := range pvs.Spec.Targets {
		if len(target.Repositories) > 0 {
			for _, rt := range target.Repositories {
				pns := []string{upstreamPackageName}
				if len(rt.PackageNames) > 0 {
					pns = rt.PackageNames
				}

				for _, pn := range pns {
					result = append(result, pvContext{
						template:       target.Template,
						repoDefault:    rt.Name,
						packageDefault: pn,
					})
				}
			}
			continue
		}

		objSel := target.ObjectSelector
		if target.RepositorySelector != nil {
			// a label selector against a set of repositories
			// equivlanet to object selector with apiVersion/kind pre-set

			objSel = &api.ObjectSelector{
				LabelSelector: *target.RepositorySelector,
				APIVersion:    configapi.TypeRepository.APIVersion(),
				Kind:          configapi.TypeRepository.Kind,
			}
		}
		// a selector against a set of arbitrary objects
		uList := &unstructured.UnstructuredList{}
		group, version := resid.ParseGroupVersion(objSel.APIVersion)
		uList.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    objSel.Kind,
		})

		opts := []client.ListOption{client.InNamespace(pvs.Namespace)}
		labelSelector, err := metav1.LabelSelectorAsSelector(&objSel.LabelSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: labelSelector})

		if err := r.Client.List(ctx, uList, opts...); err != nil {
			return nil, err
		}

		// TODO: fire event; set condition?
		if len(uList.Items) == 0 {
			klog.Warningf("no objects selected by spec.targets[%d]", i)
		}
		for _, u := range uList.Items {
			result = append(result, pvContext{
				template:       target.Template,
				repoDefault:    u.GetName(),
				packageDefault: upstreamPackageName,
				object:         u.DeepCopy(),
			})

		}
	}
	return result, nil
}

func (r *PackageVariantSetReconciler) convertObjectToRNode(obj runtime.Object) (*yaml.RNode, error) {
	var buffer bytes.Buffer
	if err := r.serializer.Encode(obj, &buffer); err != nil {
		return nil, err
	}
	return yaml.Parse(buffer.String())
}

func (r *PackageVariantSetReconciler) ensurePackageVariants(ctx context.Context, pvs *api.PackageVariantSet,
	repoList *configapi.RepositoryList, upstreamPR *porchapi.PackageRevision,
	downstreams []pvContext) error {

	var pvList pkgvarapi.PackageVariantList
	if err := r.Client.List(ctx, &pvList,
		client.InNamespace(pvs.Namespace),
		client.MatchingLabels{
			PackageVariantSetOwnerLabel: string(pvs.UID),
		}); err != nil {
		return fmt.Errorf("error listing package variants %v", err)
	}

	// existingPackageVariantMap holds the PackageVariant objects that currently exist.
	existingPackageVariantMap := make(map[string]*pkgvarapi.PackageVariant, len(pvList.Items))
	// desiredPackageVariantMap holds the PackageVariant objects that we want to exist.
	desiredPackageVariantMap := make(map[string]*pkgvarapi.PackageVariant, len(downstreams))

	for _, pv := range pvList.Items {
		pvId := packageVariantIdentifier(pvs.Name, &pv.Spec)
		existingPackageVariantMap[pvId] = pv.DeepCopy()
	}

	tr := true
	for _, downstream := range downstreams {
		pvSpec, err := renderPackageVariantSpec(ctx, pvs, repoList, upstreamPR, downstream)
		if err != nil {
			return err
		}
		pvId := packageVariantIdentifier(pvs.Name, pvSpec)
		pv := pkgvarapi.PackageVariant{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PackageVariant",
				APIVersion: "config.porch.kpt.dev",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       packageVariantName(pvId),
				Namespace:  pvs.Namespace,
				Finalizers: []string{pkgvarapi.Finalizer},
				Labels:     map[string]string{PackageVariantSetOwnerLabel: string(pvs.UID)},
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion:         pvs.APIVersion,
					Kind:               pvs.Kind,
					Name:               pvs.Name,
					UID:                pvs.UID,
					Controller:         &tr,
					BlockOwnerDeletion: nil,
				}},
			},
			Spec: *pvSpec,
		}
		desiredPackageVariantMap[pvId] = &pv
	}

	for existingPvId, existingPV := range existingPackageVariantMap {
		if _, found := desiredPackageVariantMap[existingPvId]; found {
			// this PackageVariant exists in both the desired PackageVariant set and the
			// existing PackageVariant set, so we don't need to do anything.
		} else {
			// this PackageVariant exists in the existing PackageVariant set, but not
			// the desired PackageVariant set, so we need to delete it.
			err := r.Client.Delete(ctx, existingPV)
			if err != nil {
				return err
			}
		}
	}

	for desiredPvId, desiredPv := range desiredPackageVariantMap {
		if existingPv, found := existingPackageVariantMap[desiredPvId]; found {
			// this PackageVariant exists in both the desired PackageVariant set and the
			// existing PackageVariant set, so we update it
			// we only change the spec
			existingPv.Spec = desiredPv.Spec
			err := r.Client.Update(ctx, existingPv)
			if err != nil {
				return err
			}
		} else {
			// this PackageVariant exists in the desired PackageVariant set, but not
			// the existing PackageVariant set, so we need to create it.
			err := r.Client.Create(ctx, desiredPv)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func packageVariantIdentifier(pvsName string, spec *pkgvarapi.PackageVariantSpec) string {
	return pvsName + "-" + spec.Downstream.Repo + "-" + spec.Downstream.Package
}

func packageVariantName(pvId string) string {
	if len(pvId) <= PackageVariantNameMaxLength {
		return pvId
	}

	hash := sha1.Sum([]byte(pvId))
	stubIdx := PackageVariantNameMaxLength - PackageVariantNameHashLength - 1
	return fmt.Sprintf("%s-%s", pvId[:stubIdx], hex.EncodeToString(hash[:])[:PackageVariantNameHashLength])
}

// SetupWithManager sets up the controller with the Manager.
func (r *PackageVariantSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := porchapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := configapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := pkgvarapi.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := scheme.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.Client = mgr.GetClient()
	r.serializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true})

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.PackageVariantSet{}).
		Watches(&source.Kind{Type: &pkgvarapi.PackageVariant{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectsToRequests)).
		Watches(&source.Kind{Type: &porchapi.PackageRevision{}},
			handler.EnqueueRequestsFromMapFunc(r.mapObjectsToRequests)).
		Complete(r)
}

func (r *PackageVariantSetReconciler) mapObjectsToRequests(obj client.Object) []reconcile.Request {
	attachedPackageVariants := &api.PackageVariantSetList{}
	err := r.List(context.TODO(), attachedPackageVariants, &client.ListOptions{
		Namespace: obj.GetNamespace(),
	})
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(attachedPackageVariants.Items))
	for i, item := range attachedPackageVariants.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}

func setStalledConditionsToTrue(pvs *api.PackageVariantSet, reason, message string) {
	meta.SetStatusCondition(&pvs.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeStalled,
		Status:  "True",
		Reason:  reason,
		Message: message,
	})
	meta.SetStatusCondition(&pvs.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  "False",
		Reason:  reason,
		Message: message,
	})
}
