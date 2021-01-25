// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/GoogleContainerTools/kpt/pkg/live/preprocess"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/cmd/destroy"
	"sigs.k8s.io/cli-utils/cmd/flagutils"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// GetDestroyRunner returns a wrapper around the cli-utils destroy command DestroyRunner. Sets
// up the Run on this wrapped runner to be the DestroyRunnerWrapper run.
func GetDestroyRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *DestroyRunnerWrapper {
	destroyRunner := destroy.GetDestroyRunner(provider, loader, ioStreams)
	w := &DestroyRunnerWrapper{
		destroyRunner: destroyRunner,
		provider:      provider,
	}
	// Set the wrapper run to be the RunE function for the wrapped command.
	destroyRunner.Command.RunE = w.RunE
	return w
}

// DestroyRunnerWrapper encapsulates the cli-utils destroy command DestroyRunner as well
// as structures necessary to run.
type DestroyRunnerWrapper struct {
	destroyRunner *destroy.DestroyRunner
	provider      provider.Provider
}

// Command returns the wrapped DestroyRunner cobraCommand structure.
func (w *DestroyRunnerWrapper) Command() *cobra.Command {
	return w.destroyRunner.Command
}

// RunE wraps the destroyRunner.RunE with the pre-processing for inventory policy.
func (w *DestroyRunnerWrapper) RunE(cmd *cobra.Command, args []string) error {
	if w.Command().Flag(flagutils.InventoryPolicyFlag).Value.String() == flagutils.InventoryPolicyStrict {
		w.destroyRunner.PreProcess = func(inv inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error) {
			return preprocess.PreProcess(w.provider, inv, strategy)
		}
	}
	return w.destroyRunner.RunE(cmd, args)
}
