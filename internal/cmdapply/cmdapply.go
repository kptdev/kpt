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
	"github.com/GoogleContainerTools/kpt/internal/util/strings"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/flagutils"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/printers"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		ctx:       ctx,
		Applier:   apply.NewApplier(provider),
		provider:  provider,
		ioStreams: ioStreams,
	}
	c := &cobra.Command{
		Use:     "apply [PKG_PATH | -]",
		RunE:    r.runE,
		PreRunE: r.preRunE,
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
	c.Flags().BoolVar(&r.installCRD, "install-resource-group", false,
		"If true, install the inventory ResourceGroup CRD before applying.")
	c.Flags().BoolVar(&r.dryRun, "dry-run", false,
		"dry-run apply for the resources in the package.")
	return r
}

func NewCommand(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, provider, ioStreams).Command
}

// Runner contains the run function
type Runner struct {
	ctx        context.Context
	Command    *cobra.Command
	PreProcess func(info inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error)
	ioStreams  genericclioptions.IOStreams
	Applier    *apply.Applier
	provider   provider.Provider

	installCRD                   bool
	serverSideOptions            common.ServerSideOptions
	output                       string
	period                       time.Duration
	reconcileTimeout             time.Duration
	prunePropagationPolicyString string
	pruneTimeout                 time.Duration
	inventoryPolicyString        string
	dryRun                       bool

	inventoryPolicy inventory.InventoryPolicy
	prunePropPolicy v1.DeletionPropagation
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
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

	if !r.installCRD {
		err := cmdutil.VerifyResourceGroupCRD(r.provider.Factory())
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

	objs, inv, err := live.Load(r.provider.Factory(), args[0], c.InOrStdin())
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

	if r.installCRD {
		err := cmdutil.InstallResourceGroupCRD(r.ctx, r.provider.Factory())
		if err != nil {
			return err
		}
	}

	// Run the applier. It will return a channel where we can receive updates
	// to keep track of progress and any issues.
	if err := r.Applier.Initialize(); err != nil {
		return err
	}
	ch := r.Applier.Run(r.ctx, invInfo, objs, apply.Options{
		ServerSideOptions: r.serverSideOptions,
		PollInterval:      r.period,
		ReconcileTimeout:  r.reconcileTimeout,
		// If we are not waiting for status, tell the applier to not
		// emit the events.
		EmitStatusEvents:       r.reconcileTimeout != time.Duration(0) || r.pruneTimeout != time.Duration(0),
		DryRunStrategy:         dryRunStrategy,
		PrunePropagationPolicy: r.prunePropPolicy,
		PruneTimeout:           r.pruneTimeout,
		InventoryPolicy:        r.inventoryPolicy,
	})

	// The printer will print updates from the channel. It will block
	// until the channel is closed.
	printer := printers.GetPrinter(r.output, r.ioStreams)
	return printer.Print(ch, dryRunStrategy)
}
