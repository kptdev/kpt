// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package destroy

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/cmd/flagutils"
	"sigs.k8s.io/cli-utils/cmd/printers"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// GetDestroyRunner creates and returns the DestroyRunner which stores the cobra command.
func GetDestroyRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *DestroyRunner {
	r := &DestroyRunner{
		Destroyer: apply.NewDestroyer(provider),
		ioStreams: ioStreams,
		provider:  provider,
		loader:    loader,
	}
	cmd := &cobra.Command{
		Use:                   "destroy (DIRECTORY | STDIN)",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Destroy all the resources related to configuration"),
		RunE:                  r.RunE,
	}

	cmd.Flags().StringVar(&r.output, "output", printers.DefaultPrinter(),
		fmt.Sprintf("Output format, must be one of %s", strings.Join(printers.SupportedPrinters(), ",")))
	cmd.Flags().StringVar(&r.inventoryPolicy, flagutils.InventoryPolicyFlag, flagutils.InventoryPolicyStrict,
		"It determines the behavior when the resources don't belong to current inventory. Available options "+
			fmt.Sprintf("%q and %q.", flagutils.InventoryPolicyStrict, flagutils.InventoryPolicyAdopt))

	r.Command = cmd
	return r
}

// DestroyCommand creates the DestroyRunner, returning the cobra command associated with it.
func DestroyCommand(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	provider := provider.NewProvider(f)
	loader := manifestreader.NewManifestLoader(f)
	return GetDestroyRunner(provider, loader, ioStreams).Command
}

// DestroyRunner encapsulates data necessary to run the destroy command.
type DestroyRunner struct {
	Command    *cobra.Command
	PreProcess func(info inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error)
	ioStreams  genericclioptions.IOStreams
	Destroyer  *apply.Destroyer
	provider   provider.Provider
	loader     manifestreader.ManifestLoader

	output          string
	inventoryPolicy string
}

func (r *DestroyRunner) RunE(cmd *cobra.Command, args []string) error {
	inventoryPolicy, err := flagutils.ConvertInventoryPolicy(r.inventoryPolicy)
	if err != nil {
		return err
	}

	// Retrieve the inventory object.
	reader, err := r.loader.ManifestReader(cmd.InOrStdin(), flagutils.PathFromArgs(args))
	if err != nil {
		return err
	}
	objs, err := reader.Read()
	if err != nil {
		return err
	}
	inv, _, err := r.loader.InventoryInfo(objs)
	if err != nil {
		return err
	}

	if r.PreProcess != nil {
		inventoryPolicy, err = r.PreProcess(inv, r.Destroyer.DryRunStrategy)
		if err != nil {
			return err
		}
	}

	// Run the destroyer. It will return a channel where we can receive updates
	// to keep track of progress and any issues.
	err = r.Destroyer.Initialize()
	if err != nil {
		return err
	}
	option := &apply.DestroyerOption{
		InventoryPolicy: inventoryPolicy,
	}
	ch := r.Destroyer.Run(inv, option)

	// The printer will print updates from the channel. It will block
	// until the channel is closed.
	printer := printers.GetPrinter(r.output, r.ioStreams)
	return printer.Print(ch, r.Destroyer.DryRunStrategy)
}
