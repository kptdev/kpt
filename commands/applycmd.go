// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/apply"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// Get ApplyRunner returns a wrapper around the cli-utils apply command ApplyRunner. Sets
// up the Run on this wrapped runner to be the ApplyRunnerWrapper run.
func GetApplyRunner(provider provider.Provider, loader manifestreader.ManifestLoader, ioStreams genericclioptions.IOStreams) *ApplyRunnerWrapper {
	applyRunner := apply.GetApplyRunner(provider, loader, ioStreams)
	w := &ApplyRunnerWrapper{
		applyRunner: applyRunner,
		factory:     provider.Factory(),
		ioStreams:   ioStreams,
		installCRD:  false,
	}
	cmd := applyRunner.Command
	cmd.Flags().BoolVar(&w.installCRD, "install-resource-group", false,
		"If true, install the inventory ResourceGroup CRD before applying.")
	// Set the wrapper run to be the RunE function for the wrapped command.
	applyRunner.Command.RunE = w.RunE
	return w
}

// ApplyRunnerWrapper encapsulates the cli-utils apply command ApplyRunner as well
// as structures necessary to run.
type ApplyRunnerWrapper struct {
	applyRunner *apply.ApplyRunner
	factory     cmdutil.Factory
	ioStreams   genericclioptions.IOStreams
	installCRD  bool // Install the ResourceGroup CRD before applying
}

// Command returns the wrapped ApplyRunner cobraCommand structure.
func (w *ApplyRunnerWrapper) Command() *cobra.Command {
	return w.applyRunner.Command
}

// RunE delegates to the stored applyRunner. Before the delegation, this
// function either applies the inventory ResourceGroup CRD
// (--apply-inventory-crd flag), or checks if the inventory
// ResourceGroup CRD is available. Returns an error if one occurs in the
// delegation.
func (w *ApplyRunnerWrapper) RunE(cmd *cobra.Command, args []string) error {
	// Install the inventory ResourceGroup CRD prior to applying if
	// the flag/option is present. Otherwise, check if the CRD exists
	// and report a failure.
	if w.installCRD {
		fmt.Fprint(w.ioStreams.Out, "installing inventory ResourceGroup CRD...")
		err := live.InstallResourceGroupCRD(w.factory)
		if err == nil {
			fmt.Fprintln(w.ioStreams.Out, "success")
		} else {
			fmt.Fprintln(w.ioStreams.Out, "failed")
			fmt.Fprintln(w.ioStreams.Out, "run 'kpt live install-resource-group' to try again")
			return err
		}
	} else if !live.ResourceGroupCRDApplied(w.factory) {
		// Otherwise, report the inventory ResourceGroup if missing.
		fmt.Fprintln(w.ioStreams.Out, "inventory ResourceGroup CRD is missing")
		fmt.Fprintln(w.ioStreams.Out, "run 'kpt live install-resource-group' to remedy")
		// Do NOT return here, since it breaks legacy ConfigMap applies.
	}
	return w.applyRunner.RunE(cmd, args)
}
