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

package packagevariantset

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"
	"strings"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha1"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyamlutils "sigs.k8s.io/kustomize/kyaml/utils"
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

const PackageVariantSetOwnerLabel = "config.porch.kpt.dev/packagevariantset"

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 rbac:roleName=porch-controllers-packagevariantsets webhook paths="." output:rbac:artifacts:config=../../../config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=packagevariantsets/finalizers,verbs=update

// Reconcile implements the main kubernetes reconciliation loop.
func (r *PackageVariantSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pvs, err := r.init(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	if pvs == nil {
		// maybe the pvs was deleted
		return ctrl.Result{}, nil
	}

	if errs := validatePackageVariantSet(pvs); len(errs) > 0 {
		pvs.Status.ValidationErrors = nil
		for _, validationErr := range errs {
			if validationErr.Error() != "" {
				pvs.Status.ValidationErrors = append(pvs.Status.ValidationErrors, validationErr.Error())
			}
		}
		statusUpdateErr := r.Client.Status().Update(ctx, pvs)
		return ctrl.Result{}, statusUpdateErr
	}

	upstream, err := r.getUpstreamPR(pvs.Spec.Upstream)
	if err != nil {
		return ctrl.Result{}, err
	}
	if upstream == nil {
		return ctrl.Result{}, fmt.Errorf("could not find specified upstream")
	}

	downstreams, err := r.unrollDownstreamTargets(ctx, pvs,
		upstream.Package)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.ensurePackageVariants(ctx, upstream, downstreams, pvs); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PackageVariantSetReconciler) init(ctx context.Context, req ctrl.Request) (*api.PackageVariantSet, error) {
	var pvs api.PackageVariantSet
	if err := r.Client.Get(ctx, req.NamespacedName, &pvs); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return &pvs, nil
}

func validatePackageVariantSet(pvs *api.PackageVariantSet) []error {
	var allErrs []error
	if pvs.Spec.Upstream == nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "upstream"), "{}", "missing required field"))
	} else {
		if pvs.Spec.Upstream.Package == nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "upstream", "package"), "{}", "missing required field"))
		} else {
			if pvs.Spec.Upstream.Package.Name == "" {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "upstream", "package", "name"), "", "missing required field"))
			}
			if pvs.Spec.Upstream.Package.Repo == "" {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "upstream", "package", "repo"), "", "missing required field"))
			}
		}
		if (pvs.Spec.Upstream.Tag == "" && pvs.Spec.Upstream.Revision == "") ||
			(pvs.Spec.Upstream.Tag != "" && pvs.Spec.Upstream.Revision != "") {
			allErrs = append(allErrs, fmt.Errorf("must have one of spec.upstream.revision and spec.upstream.tag"))
		}
	}

	if len(pvs.Spec.Targets) == 0 {
		allErrs = append(allErrs, fmt.Errorf("must specify at least one item in spec.targets"))
	}
	for i, target := range pvs.Spec.Targets {
		count := 0
		if target.Package != nil {
			if target.PackageName != nil {
				allErrs = append(allErrs, fmt.Errorf("spec.targets[%d] cannot specify both fields `packageName` and `package`", i))
			}
			if target.Package.Repo == "" {
				allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].package.repo cannot be empty when using `package`", i))
			}
			count++
		}
		if target.Repositories != nil {
			count++
		}
		if target.Objects != nil {
			if target.Objects.Selectors == nil {
				allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objects must have at least one selector", i))
			}
			if target.Objects.RepoName == nil {
				allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objects must specify `repoName` field", i))
			}
			for j, selector := range target.Objects.Selectors {
				if selector.APIVersion == "" {
					allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objects.selectors[%d] must specify 'apiVersion'", i, j))
				}
				if selector.Kind == "" {
					allErrs = append(allErrs, fmt.Errorf("spec.targets[%d].objects.selectors[%d] must specify 'kind'", i, j))
				}
			}
			count++
		}
		if count != 1 {
			allErrs = append(allErrs, fmt.Errorf("spec.targets[%d] must specify one of `package`, `repositories`, or `objects`", i))
		}
	}

	if pvs.Spec.AdoptionPolicy == "" {
		pvs.Spec.AdoptionPolicy = pkgvarapi.AdoptionPolicyAdoptNone
	}
	if pvs.Spec.DeletionPolicy == "" {
		pvs.Spec.DeletionPolicy = pkgvarapi.DeletionPolicyDelete
	}
	if pvs.Spec.AdoptionPolicy != pkgvarapi.AdoptionPolicyAdoptNone && pvs.Spec.AdoptionPolicy != pkgvarapi.AdoptionPolicyAdoptExisting {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "adoptionPolicy"), pvs.Spec.AdoptionPolicy,
			fmt.Sprintf("field can only be %q or %q",
				pkgvarapi.AdoptionPolicyAdoptNone, pkgvarapi.AdoptionPolicyAdoptExisting)))
	}
	if pvs.Spec.DeletionPolicy != pkgvarapi.DeletionPolicyOrphan && pvs.Spec.DeletionPolicy != pkgvarapi.DeletionPolicyDelete {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec", "deletionPolicy"), pvs.Spec.DeletionPolicy,
			fmt.Sprintf("field can only be %q or %q",
				pkgvarapi.DeletionPolicyOrphan, pkgvarapi.DeletionPolicyDelete)))
	}
	return allErrs
}

func (r *PackageVariantSetReconciler) getUpstreamPR(
	upstream *api.Upstream) (*pkgvarapi.Upstream, error) {

	// one of upstream.Tag or upstream.Revision must have been specified,
	// because this pvs has already been validated
	if upstream.Tag != "" {
		// TODO: Implement this.
		//   We need to figure out which published revision this refers to,
		//   so the controller needs to reach out to the Github repo and
		//   look at the commits on this tag.
		//   We will need to find this tag's most recent commit that refers
		//   to the relevant package, and then find the package revision
		//   that is on the same commit.
		return nil, fmt.Errorf("specifying the upstream tag is not yet supported")

	}

	// upstream.Revision is specified
	return &pkgvarapi.Upstream{
		Repo:     upstream.Package.Repo,
		Package:  upstream.Package.Name,
		Revision: upstream.Revision,
	}, nil

}

func (r *PackageVariantSetReconciler) unrollDownstreamTargets(ctx context.Context,
	pvs *api.PackageVariantSet,
	upstreamPackageName string) ([]*pkgvarapi.Downstream, error) {
	var result []*pkgvarapi.Downstream
	for _, target := range pvs.Spec.Targets {
		switch {
		case target.Package != nil:
			// an explicit repo/package name pair
			result = append(result, r.repoPackagePair(&target, upstreamPackageName))

		case target.Repositories != nil:
			// a label selector against a set of repositories
			selector, err := metav1.LabelSelectorAsSelector(target.Repositories)
			if err != nil {
				return nil, err
			}
			var repoList configapi.RepositoryList
			if err := r.Client.List(ctx, &repoList,
				client.InNamespace(pvs.Namespace),
				client.MatchingLabelsSelector{Selector: selector}); err != nil {
				return nil, err
			}
			pkgs, err := r.repositorySet(&target, upstreamPackageName, &repoList)
			if err != nil {
				return nil, fmt.Errorf("error when selecting repository set: %v", err)
			}
			result = append(result, pkgs...)

		case target.Objects != nil:
			// a selector against a set of arbitrary objects
			selectedObjects, err := r.getSelectedObjects(ctx, target.Objects.Selectors)
			if err != nil {
				return nil, err
			}
			pkgs, err := r.objectSet(&target, upstreamPackageName, selectedObjects)
			if err != nil {
				return nil, fmt.Errorf("error when selecting object set: %v", err)
			}
			result = append(result, pkgs...)
		}
	}
	return result, nil
}

func (r *PackageVariantSetReconciler) repoPackagePair(target *api.Target,
	upstreamPackageName string) *pkgvarapi.Downstream {
	downstreamPackageName := target.Package.Name
	if downstreamPackageName == "" {
		downstreamPackageName = upstreamPackageName
	}
	return &pkgvarapi.Downstream{
		Repo:    target.Package.Repo,
		Package: downstreamPackageName,
	}
}

func (r *PackageVariantSetReconciler) repositorySet(
	target *api.Target,
	upstreamPackageName string,
	repoList *configapi.RepositoryList) ([]*pkgvarapi.Downstream, error) {
	var result []*pkgvarapi.Downstream
	for _, repo := range repoList.Items {
		repoAsRNode, err := r.convertObjectToRNode(&repo)
		if err != nil {
			return nil, fmt.Errorf("error converting repo to RNode: %v", err)
		}
		downstreamPackageName, err := r.getDownstreamPackageName(target.PackageName, upstreamPackageName, repoAsRNode)
		if err != nil {
			return nil, err
		}
		result = append(result, &pkgvarapi.Downstream{
			Repo:    repo.Name,
			Package: downstreamPackageName,
		})

	}
	return result, nil
}

func (r *PackageVariantSetReconciler) objectSet(target *api.Target,
	upstreamPackageName string,
	selectedObjects map[resid.ResId]*yaml.RNode) ([]*pkgvarapi.Downstream, error) {
	var result []*pkgvarapi.Downstream
	for _, obj := range selectedObjects {
		downstreamPackageName, err := r.getDownstreamPackageName(target.PackageName,
			upstreamPackageName, obj)
		if err != nil {
			return nil, err
		}
		repo, err := r.fetchValue(target.Objects.RepoName, obj)
		if err != nil {
			return nil, err
		}
		if repo == "" {
			return nil, fmt.Errorf("error evaluating repo name: received empty string")
		}
		result = append(result, &pkgvarapi.Downstream{
			Package: downstreamPackageName,
			Repo:    repo,
		})
	}
	return result, nil
}

func (r *PackageVariantSetReconciler) getSelectedObjects(ctx context.Context, selectors []api.Selector) (map[resid.ResId]*yaml.RNode, error) {
	selectedObjects := make(map[resid.ResId]*yaml.RNode) // this is a map to prevent duplicates

	for _, selector := range selectors {
		uList := &unstructured.UnstructuredList{}
		group, version := resid.ParseGroupVersion(selector.APIVersion)
		uList.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   group,
			Version: version,
			Kind:    selector.Kind,
		})

		labelSelector, err := metav1.LabelSelectorAsSelector(selector.Labels)
		if err != nil {
			return nil, err
		}

		if err := r.Client.List(ctx, uList,
			client.InNamespace(selector.Namespace),
			client.MatchingLabelsSelector{Selector: labelSelector}); err != nil {
			return nil, fmt.Errorf("unable to list objects in cluster: %v", err)
		}

		for _, u := range uList.Items {
			objAsRNode, err := r.convertObjectToRNode(&u)
			if err != nil {
				return nil, fmt.Errorf("error converting unstructured object to RNode: %v", err)
			}
			if fnruntime.IsMatch(objAsRNode, selector.ToKptfileSelector()) {
				selectedObjects[resid.FromRNode(objAsRNode)] = objAsRNode
			}
		}
	}
	return selectedObjects, nil
}

func (r *PackageVariantSetReconciler) getDownstreamPackageName(targetName *api.PackageName,
	upstreamPackageName string,
	obj *yaml.RNode) (string, error) {

	if targetName == nil {
		return upstreamPackageName, nil
	}

	packageName, err := r.fetchValue(targetName.Name, obj)
	if err != nil {
		return "", err
	}
	if packageName == "" {
		packageName = upstreamPackageName
	}

	suffix, err := r.fetchValue(targetName.NameSuffix, obj)
	if err != nil {
		return "", err
	}

	prefix, err := r.fetchValue(targetName.NamePrefix, obj)
	if err != nil {
		return "", err
	}

	return prefix + packageName + suffix, nil
}

func (r *PackageVariantSetReconciler) convertObjectToRNode(obj runtime.Object) (*yaml.RNode, error) {
	var buffer bytes.Buffer
	if err := r.serializer.Encode(obj, &buffer); err != nil {
		return nil, err
	}
	return yaml.Parse(buffer.String())
}

func (r *PackageVariantSetReconciler) fetchValue(value *api.ValueOrFromField,
	obj *yaml.RNode) (string, error) {
	if value == nil {
		return "", nil
	}
	if value.Value != "" {
		return value.Value, nil
	}
	if value.FromField == "" {
		return "", nil
	}

	// The SmarterPathSplitter below splits on '.', and the yaml.Lookup filter expects
	// a list of path elements to parse through, e.g. ["metadata", "ownerRefs", "1"],
	// so we have to do a bit of a hack to support JSON path syntax.
	// Adding a '.' before each '[' ensures that we split our path correctly, then
	// we have to parse through and remove the [] around numbers;
	// E.g. 'metadata.ownerRefs[1]' splits into 'metadata', 'ownerRefs', '1' before
	// we call yaml.Lookup, so we first change it to 'metadata.ownerRefs.1 before calling the splitter.
	// See TestFetchValue for examples of what this supports.
	fromField := strings.ReplaceAll(value.FromField, "[", ".[")
	fieldPath := kyamlutils.SmarterPathSplitter(fromField, ".")
	for i := range fieldPath {
		trimmed := strings.Trim(fieldPath[i], "[]")
		if _, err := strconv.Atoi(trimmed); err == nil {
			fieldPath[i] = trimmed
		}
	}
	rn, err := obj.Pipe(yaml.Lookup(fieldPath...))
	if err != nil {
		return "", err
	}
	if rn.IsNilOrEmpty() || rn.YNode().Value == "" {
		return "", fmt.Errorf("value not found")
	}

	return rn.YNode().Value, nil
}

func (r *PackageVariantSetReconciler) ensurePackageVariants(ctx context.Context,
	upstream *pkgvarapi.Upstream, downstreams []*pkgvarapi.Downstream,
	pvs *api.PackageVariantSet) error {

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
		hash, err := hashFromPackageVariantSpec(&pv.Spec)
		if err != nil {
			return err
		}
		existingPackageVariantMap[hash] = &pv
	}

	tr := true
	for _, downstream := range downstreams {
		pvSpec := pkgvarapi.PackageVariantSpec{
			Upstream:       upstream,
			Downstream:     downstream,
			AdoptionPolicy: pvs.Spec.AdoptionPolicy,
			DeletionPolicy: pvs.Spec.DeletionPolicy,
			Labels:         pvs.Spec.Labels,
			Annotations:    pvs.Spec.Annotations,
		}
		hash, err := hashFromPackageVariantSpec(&pvSpec)
		if err != nil {
			return err
		}
		pv := pkgvarapi.PackageVariant{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PackageVariant",
				APIVersion: "config.porch.kpt.dev",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:       fmt.Sprintf("%s-%s", pvs.Name, hash),
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
			Spec: pvSpec,
		}
		desiredPackageVariantMap[hash] = &pv
	}

	for existingPvHash, existingPV := range existingPackageVariantMap {
		if _, found := desiredPackageVariantMap[existingPvHash]; found {
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

	for desiredPvHash, desiredPv := range desiredPackageVariantMap {
		if _, found := existingPackageVariantMap[desiredPvHash]; found {
			// this PackageVariant exists in both the desired PackageVariant set and the
			// existing PackageVariant set, so we don't need to do anything.
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

func hashFromPackageVariantSpec(spec *pkgvarapi.PackageVariantSpec) (string, error) {
	b, err := yaml.Marshal(spec)
	if err != nil {
		return "", err
	}
	hash := sha1.Sum(b)
	return hex.EncodeToString(hash[:]), nil
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
