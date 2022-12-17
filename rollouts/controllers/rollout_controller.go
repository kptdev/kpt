/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"flag"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gkeclusterapis "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/rollouts/pkg/clusterstore"
)

var (
	rootSyncNamespace = "config-management-system"
	rootSyncGVK       = schema.GroupVersionKind{
		Group:   "configsync.gke.io",
		Version: "v1beta1",
		Kind:    "RootSync",
	}
	rootSyncGVR = schema.GroupVersionResource{
		Group:    "configsync.gke.io",
		Version:  "v1beta1",
		Resource: "rootsyncs",
	}
	containerClusterKind       = "ContainerCluster"
	containerClusterApiVersion = "container.cnrm.cloud.google.com/v1beta1"

	configControllerKind       = "ConfigControllerInstance"
	configControllerApiVersion = "configcontroller.cnrm.cloud.google.com/v1beta1"
)

type Options struct {
}

func (o *Options) InitDefaults() {
}

func (o *Options) BindFlags(prefix string, flags *flag.FlagSet) {
}

// RolloutReconciler reconciles a Rollout object
type RolloutReconciler struct {
	client.Client

	store *clusterstore.ClusterStore

	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gitops.kpt.dev,resources=rollouts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Rollout object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
//
// Dumb reconciliation of Rollout API includes the following:
// Fetch the READY kcc clusters.
// For each kcc cluster, fetch RootSync objects in each of the KCC clusters.
func (r *RolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var rollout gitopsv1alpha1.Rollout

	if err := r.Get(ctx, req.NamespacedName, &rollout); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger := log.FromContext(ctx)

	logger.Info("reconciling", "key", req.NamespacedName)

	gkeClusters, err := r.store.ListClusters(ctx, rollout.Spec.Targets.Selector)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.store.PrintClusterInfos(ctx, gkeClusters)

	for _, gkeCluster := range gkeClusters.Items {
		cl, err := r.store.GetClusterClient(ctx, &gkeCluster)
		if err != nil {
			return ctrl.Result{}, err
		}
		r.testClusterClient(ctx, cl)
	}
	return ctrl.Result{}, err
}

func (r *RolloutReconciler) testClusterClient(ctx context.Context, cl client.Client) error {
	logger := log.FromContext(ctx)
	podList := &v1.PodList{}
	err := cl.List(context.Background(), podList, client.InNamespace("kube-system"))
	if err != nil {
		return err
	}
	logger.Info("found podlist", "number of pods", len(podList.Items))
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	gkeclusterapis.AddToScheme(mgr.GetScheme())

	// setup the clusterstore
	r.store = &clusterstore.ClusterStore{
		Config: mgr.GetConfig(),
		Client: r.Client,
	}
	if err := r.store.Init(); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&gitopsv1alpha1.Rollout{}).
		Complete(r)
}
