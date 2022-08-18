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

package main

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 rbac:roleName=configmanagement-operator webhook paths="./..."

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 crd paths="./..." output:crd:artifacts:config=config/crd/bases

import (
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/api/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsync/pkg/controllers/remoterootsyncset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/workloadidentitybinding/pkg/controllers/workloadidentitybinding"
	//+kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

// We include our lease / events permissions in the main RBAC role

//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(porchapi.AddToScheme(scheme))
	utilruntime.Must(api.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// var metricsAddr string
	// var enableLeaderElection bool
	// var probeAddr string

	klog.InitFlags(nil)

	// flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	// flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// flag.BoolVar(&enableLeaderElection, "leader-elect", false,
	// 	"Enable leader election for controller manager. "+
	// 		"Enabling this will ensure there is only one active controller manager.")

	managerOptions := ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         ":8080",
		Port:                       9443,
		HealthProbeBindAddress:     ":8081",
		LeaderElection:             false,
		LeaderElectionID:           "porch-operators.config.cloud.google.com",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
	}

	flag.Parse()

	ctrl.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		return fmt.Errorf("error creating manager: %w", err)
	}

	if err = (&remoterootsyncset.RemoteRootSyncSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("error creating RemoteRootSyncSetReconciler controller: %w", err)
	}

	if err = (&workloadidentitybinding.WorkloadIdentityBindingReconciler{}).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("error creating WorkloadIdentityBindingReconciler controller: %w", err)
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("error adding health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("error adding ready check: %w", err)
	}

	klog.Infof("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("error running manager: %w", err)
	}
	return nil
}
