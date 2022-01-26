// Copyright 2021 Google LLC
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

package cmdapply

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/strings"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/pkg/status"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/flagutils"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/printers"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, factory util.Factory,
	ioStreams genericclioptions.IOStreams) *Runner {

	r := &Runner{
		ctx:         ctx,
		ioStreams:   ioStreams,
		factory:     factory,
		applyRunner: runApply,
	}
	c := &cobra.Command{
		Use:     "apply [PKG_PATH | -]",
		RunE:    r.runE,
		PreRunE: r.preRunE,
		Short:   livedocs.ApplyShort,
		Long:    livedocs.ApplyShort + "\n" + livedocs.ApplyLong,
		Example: livedocs.ApplyExamples,
	}
	r.Command = c

	c.Flags().BoolVar(&r.serverSideOptions.ServerSideApply, "server-side", false,
		"If true, apply merge patch is calculated on API server instead of client.")
	c.Flags().BoolVar(&r.serverSideOptions.ForceConflicts, "force-conflicts", false,
		"If true, overwrite applied fields on server if field manager conflict.")
	c.Flags().StringVar(&r.serverSideOptions.FieldManager, "field-manager", common.DefaultFieldManager,
		"The client owner of the fields being applied on the server-side.")
	c.Flags().StringVar(&r.output, "output", printers.DefaultPrinter(),
		fmt.Sprintf("Output format, must be one of %s", strings.JoinStringsWithQuotes(printers.SupportedPrinters())))
	c.Flags().DurationVar(&r.period, "poll-period", 2*time.Second,
		"Polling period for resource statuses.")
	c.Flags().DurationVar(&r.reconcileTimeout, "reconcile-timeout", time.Duration(0),
		"Timeout threshold for waiting for all resources to reach the Current status.")
	c.Flags().StringVar(&r.prunePropagationPolicyString, "prune-propagation-policy",
		"Background", "Propagation policy for pruning")
	c.Flags().DurationVar(&r.pruneTimeout, "prune-timeout", time.Duration(0),
		"Timeout threshold for waiting for all pruned resources to be deleted")
	c.Flags().StringVar(&r.inventoryPolicyString, flagutils.InventoryPolicyFlag, flagutils.InventoryPolicyStrict,
		"It determines the behavior when the resources don't belong to current inventory. Available options "+
			fmt.Sprintf("%q and %q.", flagutils.InventoryPolicyStrict, flagutils.InventoryPolicyAdopt))
	c.Flags().BoolVar(&r.installCRD, "install-resource-group", true,
		"If true, install the inventory ResourceGroup CRD before applying.")
	c.Flags().BoolVar(&r.dryRun, "dry-run", false,
		"dry-run apply for the resources in the package.")
	c.Flags().BoolVar(&r.printStatusEvents, "show-status-events", false,
		"Print status events (always enabled for table output)")
	return r
}

func NewCommand(ctx context.Context, factory util.Factory,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, factory, ioStreams).Command
}

// Runner contains the run function
type Runner struct {
	ctx        context.Context
	Command    *cobra.Command
	PreProcess func(info inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error)
	ioStreams  genericclioptions.IOStreams
	factory    util.Factory

	installCRD                   bool
	serverSideOptions            common.ServerSideOptions
	output                       string
	period                       time.Duration
	reconcileTimeout             time.Duration
	prunePropagationPolicyString string
	pruneTimeout                 time.Duration
	inventoryPolicyString        string
	dryRun                       bool
	printStatusEvents            bool

	inventoryPolicy inventory.InventoryPolicy
	prunePropPolicy v1.DeletionPropagation

	applyRunner func(r *Runner, invInfo inventory.InventoryInfo, objs []*unstructured.Unstructured,
		dryRunStrategy common.DryRunStrategy) error
}

func (r *Runner) preRunE(cmd *cobra.Command, _ []string) error {
	var err error
	r.prunePropPolicy, err = flagutils.ConvertPropagationPolicy(r.prunePropagationPolicyString)
	if err != nil {
		return err
	}

	r.inventoryPolicy, err = flagutils.ConvertInventoryPolicy(r.inventoryPolicyString)
	if err != nil {
		return err
	}

	if found := printers.ValidatePrinterType(r.output); !found {
		return fmt.Errorf("unknown output type %q", r.output)
	}

	// We default the install-resource-group flag to false if we are doing
	// dry-run, unless the user has explicitly used the install-resource-group flag.
	if r.dryRun && !cmd.Flags().Changed("install-resource-group") {
		r.installCRD = false
	}

	if !r.installCRD {
		err := cmdutil.VerifyResourceGroupCRD(r.factory)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = append(args, cwd)
	}
	path := args[0]
	var err error
	if args[0] != "-" {
		path, err = argutil.ResolveSymlink(r.ctx, path)
		if err != nil {
			return err
		}
	}

	objs, inv, err := live.Load(r.factory, path, c.InOrStdin())
	if err != nil {
		return err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return err
	}

	dryRunStrategy := common.DryRunNone
	if r.dryRun {
		if r.serverSideOptions.ServerSideApply {
			dryRunStrategy = common.DryRunServer
		} else {
			dryRunStrategy = common.DryRunClient
		}
	}

	// TODO(mortent): Figure out if we can do this differently.
	if r.PreProcess != nil {
		r.inventoryPolicy, err = r.PreProcess(invInfo, dryRunStrategy)
		if err != nil {
			return err
		}
	}

	return r.applyRunner(r, invInfo, objs, dryRunStrategy)
}

func runApply(r *Runner, invInfo inventory.InventoryInfo, objs []*unstructured.Unstructured,
	dryRunStrategy common.DryRunStrategy) error {
	if r.installCRD {
		f := r.factory
		// Only install the ResourceGroup CRD if it is not already installed.
		if err := cmdutil.VerifyResourceGroupCRD(f); err != nil {
			err = cmdutil.InstallResourceGroupCRD(r.ctx, f)
			if err != nil {
				return err
			}
		}
	}

	// Run the applier. It will return a channel where we can receive updates
	// to keep track of progress and any issues.
	invClient, err := inventory.NewInventoryClient(r.factory, live.WrapInventoryObj, live.InvToUnstructuredFunc)
	if err != nil {
		return err
	}

	statusPoller, err := status.NewStatusPoller(r.factory)
	if err != nil {
		return err
	}

	applier, err := apply.NewApplier(r.factory, invClient)
	if err != nil {
		return err
	}
	// TODO(mortent): See if we can improve this. Having to change the Applier after it has been
	// created feels a bit awkward.
	applier.StatusPoller = statusPoller

	ch := applier.Run(r.ctx, invInfo, objs, apply.Options{
		ServerSideOptions:      r.serverSideOptions,
		PollInterval:           r.period,
		ReconcileTimeout:       r.reconcileTimeout,
		EmitStatusEvents:       true, // We are always waiting for reconcile.
		DryRunStrategy:         dryRunStrategy,
		PrunePropagationPolicy: r.prunePropPolicy,
		PruneTimeout:           r.pruneTimeout,
		InventoryPolicy:        r.inventoryPolicy,
	})

	// Print the preview strategy unless the output format is json.
	if dryRunStrategy.ClientOrServerDryRun() && r.output != printers.JSONPrinter {
		if dryRunStrategy.ServerDryRun() {
			fmt.Println("Dry-run strategy: server")
		} else {
			fmt.Println("Dry-run strategy: client")
		}
	}

	// The printer will print updates from the channel. It will block
	// until the channel is closed.
	printer := printers.GetPrinter(r.output, r.ioStreams)
	return printer.Print(ch, dryRunStrategy, r.printStatusEvents)
}
