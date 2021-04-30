// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package preview

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/flagutils"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/printers"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

var (
	noPrune        = false
	previewDestroy = false
)

// GetPreviewRunner creates and returns the PreviewRunner which stores the cobra command.
func GetPreviewRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *PreviewRunner {
	r := &PreviewRunner{
		Applier:   apply.NewApplier(provider),
		Destroyer: apply.NewDestroyer(provider),
		ioStreams: ioStreams,
		provider:  provider,
		loader:    loader,
	}
	cmd := &cobra.Command{
		Use:                   "preview [PKG_PATH | -]",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Preview the apply of a configuration"),
		Args:                  cobra.MaximumNArgs(1),
		RunE:                  r.RunE,
	}

	cmd.Flags().BoolVar(&noPrune, "no-prune", noPrune, "If true, do not prune previously applied objects.")
	cmd.Flags().BoolVar(&r.serverSideOptions.ServerSideApply, "server-side", false,
		"If true, preview runs in the server instead of the client.")
	cmd.Flags().BoolVar(&r.serverSideOptions.ForceConflicts, "force-conflicts", false,
		"If true during server-side preview, do not report field conflicts.")
	cmd.Flags().StringVar(&r.serverSideOptions.FieldManager, "field-manager", common.DefaultFieldManager,
		"If true during server-side preview, sets field owner.")
	cmd.Flags().BoolVar(&previewDestroy, "destroy", previewDestroy, "If true, preview of destroy operations will be displayed.")
	cmd.Flags().StringVar(&r.output, "output", printers.DefaultPrinter(),
		fmt.Sprintf("Output format, must be one of %s", strings.Join(printers.SupportedPrinters(), ",")))
	cmd.Flags().StringVar(&r.inventoryPolicy, flagutils.InventoryPolicyFlag, flagutils.InventoryPolicyStrict,
		"It determines the behavior when the resources don't belong to current inventory. Available options "+
			fmt.Sprintf("%q and %q.", flagutils.InventoryPolicyStrict, flagutils.InventoryPolicyAdopt))

	r.Command = cmd
	return r
}

// PreviewCommand creates the PreviewRunner, returning the cobra command associated with it.
func PreviewCommand(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	provider := provider.NewProvider(f)
	loader := manifestreader.NewManifestLoader(f)
	return GetPreviewRunner(provider, loader, ioStreams).Command
}

// PreviewRunner encapsulates data necessary to run the preview command.
type PreviewRunner struct {
	Command    *cobra.Command
	PreProcess func(info inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error)
	ioStreams  genericclioptions.IOStreams
	Applier    *apply.Applier
	Destroyer  *apply.Destroyer
	provider   provider.Provider
	loader     manifestreader.ManifestLoader

	serverSideOptions common.ServerSideOptions
	output            string
	inventoryPolicy   string
}

// RunE is the function run from the cobra command.
func (r *PreviewRunner) RunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// default to the current working directory
		args = append(args, ".")
	}
	var ch <-chan event.Event

	drs := common.DryRunClient
	if r.serverSideOptions.ServerSideApply {
		drs = common.DryRunServer
	}

	if previewDestroy {
		r.Destroyer.DryRunStrategy = drs
	}

	inventoryPolicy, err := flagutils.ConvertInventoryPolicy(r.inventoryPolicy)
	if err != nil {
		return err
	}

	reader, err := r.loader.ManifestReader(cmd.InOrStdin(), flagutils.PathFromArgs(args))
	if err != nil {
		return err
	}
	objs, err := reader.Read()
	if err != nil {
		return err
	}

	inv, objs, err := r.loader.InventoryInfo(objs)
	if err != nil {
		return err
	}

	if r.PreProcess != nil {
		inventoryPolicy, err = r.PreProcess(inv, drs)
		if err != nil {
			return err
		}
	}

	// if destroy flag is set in preview, transmit it to destroyer DryRunStrategy flag
	// and pivot execution to destroy with dry-run
	if !r.Destroyer.DryRunStrategy.ClientOrServerDryRun() {
		err = r.Applier.Initialize()
		if err != nil {
			return err
		}

		// Create a context
		ctx := context.Background()

		_, err := common.DemandOneDirectory(args)
		if err != nil {
			return err
		}

		// Run the applier. It will return a channel where we can receive updates
		// to keep track of progress and any issues.
		ch = r.Applier.Run(ctx, inv, objs, apply.Options{
			EmitStatusEvents:  false,
			NoPrune:           noPrune,
			DryRunStrategy:    drs,
			ServerSideOptions: r.serverSideOptions,
			InventoryPolicy:   inventoryPolicy,
		})
	} else {
		err = r.Destroyer.Initialize()
		if err != nil {
			return err
		}
		option := &apply.DestroyerOption{
			InventoryPolicy: inventoryPolicy,
		}
		ch = r.Destroyer.Run(inv, option)
	}

	// The printer will print updates from the channel. It will block
	// until the channel is closed.
	printer := printers.GetPrinter(r.output, r.ioStreams)
	return printer.Print(ch, drs)
}
