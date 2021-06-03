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

package cmddestroy

import (
	"context"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/flagutils"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/printers"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

func NewRunner(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *Runner {

	r := &Runner{
		ctx:           ctx,
		Destroyer:     apply.NewDestroyer(provider),
		provider:      provider,
		ioStreams:     ioStreams,
		destroyRunner: runDestroy,
	}
	c := &cobra.Command{
		Use:     "destroy [PKG_PATH | -]",
		RunE:    r.runE,
		PreRunE: r.preRunE,
		Short:   livedocs.DestroyShort,
		Long:    livedocs.DestroyShort + "\n" + livedocs.DestroyLong,
		Example: livedocs.DestroyExamples,
	}
	r.Command = c

	c.Flags().StringVar(&r.output, "output", printers.DefaultPrinter(),
		fmt.Sprintf("Output format, must be one of %s", cmdutil.JoinStringsWithQuotes(printers.SupportedPrinters())))
	c.Flags().StringVar(&r.inventoryPolicyString, flagutils.InventoryPolicyFlag, flagutils.InventoryPolicyStrict,
		"It determines the behavior when the resources don't belong to current inventory. Available options "+
			fmt.Sprintf("%q and %q.", flagutils.InventoryPolicyStrict, flagutils.InventoryPolicyAdopt))
	c.Flags().BoolVar(&r.dryRun, "dry-run", false,
		"dry-run apply for the resources in the package.")
	return r
}

// NewCommand returns a cobra command.
func NewCommand(ctx context.Context, provider provider.Provider,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(ctx, provider, ioStreams).Command
}

// Runner contains the run function that contains the cli functionality for the
// destroy command.
type Runner struct {
	ctx        context.Context
	Command    *cobra.Command
	PreProcess func(info inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error)
	Destroyer  *apply.Destroyer
	provider   provider.Provider
	ioStreams  genericclioptions.IOStreams

	output                string
	inventoryPolicyString string
	dryRun                bool

	inventoryPolicy inventory.InventoryPolicy

	// TODO(mortent): This is needed for now since we don't have a good way to
	// stub out the Destroyer with an interface for testing purposes.
	destroyRunner func(r *Runner, inv inventory.InventoryInfo, strategy common.DryRunStrategy) error
}

// preRunE validates the inventoryPolicy and the output type.
func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	var err error
	r.inventoryPolicy, err = flagutils.ConvertInventoryPolicy(r.inventoryPolicyString)
	if err != nil {
		return err
	}

	if found := printers.ValidatePrinterType(r.output); !found {
		return fmt.Errorf("unknown output type %q", r.output)
	}

	return nil
}

// runE handles the input flags and args, sets up the Destroyer, and
// invokes the
func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// default to the current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		args = append(args, cwd)
	}

	_, inv, err := live.Load(r.provider.Factory(), args[0], c.InOrStdin())
	if err != nil {
		return err
	}

	invInfo, err := live.ToInventoryInfo(inv)
	if err != nil {
		return err
	}

	dryRunStrategy := common.DryRunNone
	if r.dryRun {
		dryRunStrategy = common.DryRunClient
	}

	// TODO(mortent): Figure out if we can do this differently.
	if r.PreProcess != nil {
		r.inventoryPolicy, err = r.PreProcess(invInfo, dryRunStrategy)
		if err != nil {
			return err
		}
	}

	return r.destroyRunner(r, invInfo, dryRunStrategy)
}

func runDestroy(r *Runner, inv inventory.InventoryInfo, dryRunStrategy common.DryRunStrategy) error {
	// Run the destroyer. It will return a channel where we can receive updates
	// to keep track of progress and any issues.
	err := r.Destroyer.Initialize()
	if err != nil {
		return err
	}
	option := &apply.DestroyerOption{
		InventoryPolicy: r.inventoryPolicy,
		DryRunStrategy:  dryRunStrategy,
	}
	ch := r.Destroyer.Run(inv, option)

	// The printer will print updates from the channel. It will block
	// until the channel is closed.
	printer := printers.GetPrinter(r.output, r.ioStreams)
	return printer.Print(ch, dryRunStrategy)
}
