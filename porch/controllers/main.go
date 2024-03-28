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

package main

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 rbac:roleName=porch-controllers webhook paths="."

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 crd paths="./..." output:crd:artifacts:config=config/crd/bases

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"golang.org/x/exp/slices"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/GoogleContainerTools/kpt/porch/controllers/fleetsyncs/pkg/controllers/fleetsync"
	"github.com/GoogleContainerTools/kpt/porch/controllers/functiondiscovery"
	"github.com/GoogleContainerTools/kpt/porch/controllers/klippy/pkg/controllers/klippy"
	"github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/pkg/controllers/packagevariant"
	"github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/pkg/controllers/packagevariantset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/remoterootsyncsets/pkg/controllers/remoterootsyncset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncdeployments/pkg/controllers/rootsyncdeployment"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncrollouts/pkg/controllers/rootsyncrollout"
	"github.com/GoogleContainerTools/kpt/porch/controllers/rootsyncsets/pkg/controllers/rootsyncset"
	"github.com/GoogleContainerTools/kpt/porch/controllers/workloadidentitybindings/pkg/controllers/workloadidentitybinding"
	"github.com/GoogleContainerTools/kpt/porch/pkg/controllerrestmapper"
	//+kubebuilder:scaffold:imports
)

var (
	reconcilers = map[string]Reconciler{
		"packagevariants":          &packagevariant.PackageVariantReconciler{},
		"packagevariantsets":       &packagevariantset.PackageVariantSetReconciler{},
		"rootsyncsets":             &rootsyncset.RootSyncSetReconciler{},
		"remoterootsyncsets":       &remoterootsyncset.RemoteRootSyncSetReconciler{},
		"workloadidentitybindings": &workloadidentitybinding.WorkloadIdentityBindingReconciler{},
		"klippy":                   &klippy.KlippyReconciler{},
		"rootsyncdeployments":      rootsyncdeployment.NewRootSyncDeploymentReconciler(),
		"functiondiscovery":        &functiondiscovery.FunctionReconciler{},
		"rootsyncrollouts":         rootsyncrollout.NewRootSyncRolloutReconciler(),
		"fleetsyncs":               fleetsync.NewFleetSyncReconciler(),
	}
)

// Reconciler is the interface implemented by (our) reconcilers, which includes some configuration and initialization.
type Reconciler interface {
	reconcile.Reconciler

	// InitDefaults populates default values into our options
	InitDefaults()

	// BindFlags binds options to flags
	BindFlags(prefix string, flags *flag.FlagSet)

	// SetupWithManager registers the reconciler to run under the specified manager
	SetupWithManager(ctrl.Manager) error
}

// We include our lease / events permissions in the main RBAC role

//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

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
	var enabledReconcilersString string

	for _, reconciler := range reconcilers {
		reconciler.InitDefaults()
	}

	klog.InitFlags(nil)

	// flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	// flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	// flag.BoolVar(&enableLeaderElection, "leader-elect", false,
	// 	"Enable leader election for controller manager. "+
	// 		"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&enabledReconcilersString, "reconcilers", "", "reconcilers that should be enabled; use * to mean 'enable all'")

	for name, reconciler := range reconcilers {
		reconciler.BindFlags(name+".", flag.CommandLine)
	}

	flag.Parse()

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unexpected additional (non-flag) arguments: %v", flag.Args())
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("error initializing scheme: %w", err)
	}

	managerOptions := ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         ":8080",
		Port:                       9443,
		HealthProbeBindAddress:     ":8081",
		LeaderElection:             false,
		LeaderElectionID:           "porch-operators.config.porch.kpt.dev",
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		MapperProvider:             controllerrestmapper.New,
	}

	ctrl.SetLogger(klogr.New())

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), managerOptions)
	if err != nil {
		return fmt.Errorf("error creating manager: %w", err)
	}

	enabledReconcilers := parseReconcilers(enabledReconcilersString)
	var enabled []string
	for name, reconciler := range reconcilers {
		if !reconcilerIsEnabled(enabledReconcilers, name) {
			continue
		}
		if err = reconciler.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("error creating %s reconciler: %w", name, err)
		}
		enabled = append(enabled, name)
	}

	if len(enabled) == 0 {
		klog.Warningf("no reconcilers are enabled; did you forget to pass the --reconcilers flag?")
	} else {
		klog.Infof("enabled reconcilers: %v", strings.Join(enabled, ","))
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

func parseReconcilers(reconcilers string) []string {
	return strings.Split(reconcilers, ",")
}

func reconcilerIsEnabled(reconcilers []string, reconciler string) bool {
	if slices.Contains(reconcilers, "*") {
		return true
	}
	if slices.Contains(reconcilers, reconciler) {
		return true
	}
	if _, found := os.LookupEnv(fmt.Sprintf("ENABLE_%s", strings.ToUpper(reconciler))); found {
		return true
	}
	return false
}
