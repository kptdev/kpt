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

package klippy

import (
	"context"
	"flag"
	"fmt"
	"strings"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/applyset"
	"github.com/GoogleContainerTools/kpt/porch/pkg/objects"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Options struct {
	// BindFunction is the image name for the "bind" function
	BindFunction string
}

func (o *Options) InitDefaults() {
	o.BindFunction = "bind"
}

func (o *Options) BindFlags(prefix string, flagset *flag.FlagSet) {
	flagset.StringVar(&o.BindFunction, prefix+"bindFunction", o.BindFunction, "image name for the bind function")
}

// KlippyReconciler reconciles Klippy objects
type KlippyReconciler struct {
	Options

	client client.Client

	dynamicClient dynamic.Interface
	restMapper    meta.RESTMapper
}

//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisionresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisionresources/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=porch.kpt.dev,resources=packagerevisions/status,verbs=get;update;patch

// Reconcile implements the main kubernetes reconciliation loop.
func (r *KlippyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("reconciling object", "id", req.NamespacedName)

	var parent api.PackageRevision
	if err := r.client.Get(ctx, req.NamespacedName, &parent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: Can we filter the watch?
	if !parent.Status.Deployment {
		log.V(2).Info("ignoring package not in deployments repository", "package", parent.Spec.PackageName)
		return ctrl.Result{}, nil
	}

	var parentResources api.PackageRevisionResources
	if err := r.client.Get(ctx, req.NamespacedName, &parentResources); err != nil {
		// Not found here is unexpected
		return ctrl.Result{}, err
	}

	if err := r.reconcile(ctx, &parent, &parentResources); err != nil {
		// TODO: raise event?
		log.Error(err, "error reconciling", "id", req.NamespacedName)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KlippyReconciler) reconcile(ctx context.Context, parent *api.PackageRevision, parentResources *api.PackageRevisionResources) error {
	log := log.FromContext(ctx)

	var blueprintPackageRevisions api.PackageRevisionList
	if err := r.client.List(ctx, &blueprintPackageRevisions); err != nil {
		return fmt.Errorf("error listing blueprints: %w", err)
	}

	// TODO: We should be able to cache this
	blueprints, err := r.parseBlueprints(ctx, &blueprintPackageRevisions)
	if err != nil {
		return err
	}

	parentObjects, err := objects.Parser{}.AsUnstructureds(parentResources.Spec.Resources)
	if err != nil {
		return err
	}

	var allProposals []*api.PackageRevision
	for _, blueprint := range blueprints {
		proposals, err := r.buildProposals(ctx, parent, parentObjects, blueprint)
		if err != nil {
			return err
		}
		allProposals = append(allProposals, proposals...)
	}

	log.Info("built proposals", "proposal", allProposals)
	if err := r.storeProposals(ctx, parent, allProposals); err != nil {
		return err
	}

	return nil
}

func (r *KlippyReconciler) parseBlueprints(ctx context.Context, packageRevisions *api.PackageRevisionList) ([]*blueprint, error) {
	var blueprints []*blueprint

	for i := range packageRevisions.Items {
		packageRevision := &packageRevisions.Items[i]

		// Only match blueprint packages
		// TODO: Push-down into a field selector?
		if packageRevision.Status.Deployment {
			continue
		}

		// TODO: Cache

		var packageRevisionResources api.PackageRevisionResources
		id := types.NamespacedName{
			Namespace: packageRevision.Namespace,
			Name:      packageRevision.Name,
		}
		if err := r.client.Get(ctx, id, &packageRevisionResources); err != nil {
			return nil, fmt.Errorf("error fetching PackageRevisionResources %v: %w", id, err)
		}

		objects, err := objects.Parser{}.AsUnstructureds(packageRevisionResources.Spec.Resources)
		if err != nil {
			return nil, err
		}

		blueprint := &blueprint{
			Objects:     objects,
			ID:          id,
			PackageName: packageRevision.Spec.PackageName,
		}
		blueprints = append(blueprints, blueprint)
	}
	return blueprints, nil
}

type bindingSlotRequirements struct {
	MatchLabels map[string]string
}

type blueprint struct {
	ID          types.NamespacedName
	PackageName string
	Objects     []*unstructured.Unstructured
}

func (r *KlippyReconciler) buildProposals(ctx context.Context, parent *api.PackageRevision, parentObjects []*unstructured.Unstructured, blueprint *blueprint) ([]*api.PackageRevision, error) {
	log := log.FromContext(ctx)

	var proposals []*api.PackageRevision

	slots := make(map[schema.GroupKind]*bindingSlotRequirements)
	for _, obj := range blueprint.Objects {
		annotations := obj.GetAnnotations()
		if annotations["config.kubernetes.io/local-config"] != "binding" {
			continue
		}

		gk := obj.GroupVersionKind().GroupKind()

		requirements := &bindingSlotRequirements{}
		requirements.MatchLabels = obj.GetLabels()

		slots[gk] = requirements
	}

	if len(slots) == 0 {
		// Though technically this is "a match", it tends to be incorrect and leads to over-proposing
		return nil, nil
	}

	isMatch := false
	bindings := make(map[schema.GroupKind]*unstructured.Unstructured)
	{
		for _, obj := range parentObjects {
			annotations := obj.GetAnnotations()
			if annotations["config.kubernetes.io/local-config"] == "binding" {
				// Ignore binding objects in the "parent"; they don't satisfy bindings
				continue
			}

			gk := obj.GroupVersionKind().GroupKind()

			requirements, found := slots[gk]
			if !found {
				continue
			}

			matchesLabels := true
			for k, v := range requirements.MatchLabels {
				if obj.GetLabels()[k] != v {
					matchesLabels = false
				}
			}
			if !matchesLabels {
				continue
			}
			bindings[gk] = obj
		}

		isMatch = len(bindings) == len(slots)
		if !isMatch {
			log.Info("package does not match", "parent", parent.Spec.PackageName, "child", blueprint.PackageName)
		}
	}

	if isMatch {
		log.Info("matched package", "parent", parent.Spec.PackageName, "child", blueprint.PackageName)
		// TODO: How to avoid collisions
		name := "packagename-" + strings.ReplaceAll(blueprint.PackageName, "/", "-")
		packageName := blueprint.PackageName

		clone := &api.PackageCloneTaskSpec{}
		clone.Upstream = api.UpstreamPackage{
			UpstreamRef: &api.PackageRevisionRef{
				Name: blueprint.ID.Name,
			},
		}

		parentName := parent.Spec.PackageName
		objectName := packageName + "-" + parentName + "-" + name

		// TODO: How to sanitize?
		// packageName can be something like dir/name, and that isn't allowed in names
		objectName = strings.ReplaceAll(objectName, "/", "-")

		proposal := &api.PackageRevision{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PackageRevision",
				APIVersion: api.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: parent.GetNamespace(),
				Name:      objectName,
			},
			Spec: api.PackageRevisionSpec{
				PackageName:    name,
				Revision:       "v1",
				RepositoryName: parent.Spec.RepositoryName,
				// TODO: Speculative: true,
				Tasks: []api.Task{
					{
						Type:  api.TaskTypeClone,
						Clone: clone,
					},
				},
			},
		}
		proposal.Labels = map[string]string{
			"alpha.kpt.dev/proposal": "",
		}
		proposal.Spec.Parent = &api.ParentReference{
			Name: parent.Name,
		}

		// TODO: Share the workspace?  Create a new one?
		proposal.Spec.WorkspaceName = parent.Spec.WorkspaceName

		for _, bindingObj := range bindings {
			bindingCore := &unstructured.Unstructured{}
			bindingCore.SetAPIVersion(bindingObj.GetAPIVersion())
			bindingCore.SetKind(bindingObj.GetKind())
			bindingCore.SetName(bindingObj.GetName())
			if bindingObj.GetNamespace() != "" {
				bindingCore.SetNamespace(bindingObj.GetNamespace())
			}

			var bindingConfig runtime.RawExtension
			bindingConfig.Object = bindingCore

			image := r.Options.BindFunction

			proposal.Spec.Tasks = append(proposal.Spec.Tasks, api.Task{
				Type: api.TaskTypeEval,
				Eval: &api.FunctionEvalTaskSpec{
					Image:  image,
					Config: bindingConfig,
				},
			})

		}

		proposals = append(proposals, proposal)
	}

	return proposals, nil
}

func (r *KlippyReconciler) storeProposals(ctx context.Context, parent *api.PackageRevision, children []*api.PackageRevision) error {
	log := log.FromContext(ctx)

	// TODO: Cache applyset

	// TODO: Should the fieldmanager just be klippy?  These objects should be owned
	patchOptions := metav1.PatchOptions{
		FieldManager: "klippy-" + parent.GetNamespace() + "-" + parent.GetName(),
	}

	// We force to overcome errors like: Apply failed with 1 conflict: conflict with "kubectl-client-side-apply" using apps/v1: .spec.template.spec.containers[name="porch-server"].image
	// TODO: How to handle this better
	force := true
	patchOptions.Force = &force

	applier, err := applyset.New(applyset.Options{
		RESTMapper:   r.restMapper,
		Client:       r.dynamicClient,
		PatchOptions: patchOptions,
	})
	if err != nil {
		return err
	}

	// TODO: Set owner refs

	var applyableObjects []applyset.ApplyableObject
	for _, o := range children {
		applyableObjects = append(applyableObjects, o)
	}
	if err := applier.ReplaceAllObjects(applyableObjects); err != nil {
		return err
	}

	results, err := applier.ApplyOnce(ctx)
	if err != nil {
		return fmt.Errorf("failed to apply proposals: %w", err)
	}

	// TODO: Signal that we don't care about health?

	log.Info("applied objects", "results", results)

	// TODO: Implement pruning

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KlippyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.client = mgr.GetClient()

	r.restMapper = mgr.GetRESTMapper()

	restConfig := mgr.GetConfig()

	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create a new dynamic client: %w", err)
	}
	r.dynamicClient = client

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&api.PackageRevisionResources{}).
		Complete(r); err != nil {
		return err
	}

	return nil
}
