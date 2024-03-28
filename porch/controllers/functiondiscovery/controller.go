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

package functiondiscovery

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"time"

	kptoci "github.com/GoogleContainerTools/kpt/pkg/oci"
	api "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/applyset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

// FunctionReconciler creates Function objects
type FunctionReconciler struct {
	Options

	client     client.Client
	restConfig *rest.Config

	ociStorage *kptoci.Storage
}

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers-functions webhook paths="." output:rbac:artifacts:config=config/rbac

//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=repositories,verbs=get;list;watch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.porch.kpt.dev,resources=functions/finalizers,verbs=update

// Reconcile implements the main kubernetes reconciliation loop.
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var subject api.Repository
	if err := r.client.Get(ctx, req.NamespacedName, &subject); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	applyResults, err := r.reconcileRepository(ctx, &subject)
	if err != nil {
		// TODO: Update status?
		// TODO: Post event?
		return ctrl.Result{}, err
	}

	if applyResults != nil && !(applyResults.AllApplied() && applyResults.AllHealthy()) {
		return ctrl.Result{Requeue: true}, nil
	}

	// Poll the repo every 15 minutes (with some jitter)
	jitter := time.Duration(rand.Intn(60)) * time.Second
	pollInterval := 15*time.Minute + jitter
	return ctrl.Result{RequeueAfter: pollInterval}, nil
}

func (r *FunctionReconciler) reconcileRepository(ctx context.Context, subject *api.Repository) (*applyset.ApplyResults, error) {
	// TODO: Cache dynamicClient / discovery etc
	restConfig := r.restConfig

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

	objects, err := r.buildObjectsToApply(ctx, subject)
	if err != nil {
		return nil, err
	}

	// TODO: Cache applyset
	patchOptions := metav1.PatchOptions{
		FieldManager: "functions-" + subject.GetNamespace() + "-" + subject.GetName(),
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
		return nil, fmt.Errorf("failed to apply: %w", err)
	}

	// TODO: Implement pruning

	return results, nil
}

// buildObjectsToApply discovers the functions in the repository.
func (r *FunctionReconciler) buildObjectsToApply(ctx context.Context, subject *api.Repository) ([]applyset.ApplyableObject, error) {
	functions, err := r.listFunctions(ctx, subject)
	if err != nil {
		return nil, err
	}

	var applyables []applyset.ApplyableObject
	for _, function := range functions {
		function.APIVersion = api.TypeFunction.APIVersion()
		function.Kind = api.TypeFunction.Kind
		applyables = append(applyables, function)
	}
	return applyables, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := api.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	r.restConfig = mgr.GetConfig()
	r.client = mgr.GetClient()

	if err := ctrl.NewControllerManagedBy(mgr).
		For(&api.Repository{}).
		Complete(r); err != nil {
		return err
	}

	cacheDir := "./.cache"

	ociStorage, err := kptoci.NewStorage(cacheDir)
	if err != nil {
		return err
	}

	r.ociStorage = ociStorage

	return nil
}
