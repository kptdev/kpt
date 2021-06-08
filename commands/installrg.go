// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"io"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// InstallRGRunner encapsulates fields for the kpt live install-resource-group command.
type InstallRGRunner struct {
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams
	factory   cmdutil.Factory
}

// GetInstallRGRunner returns a pointer to an initial InstallRGRunner structure.
func GetInstallRGRunner(factory cmdutil.Factory, ioStreams genericclioptions.IOStreams) *InstallRGRunner {
	r := &InstallRGRunner{
		factory:   factory,
		ioStreams: ioStreams,
	}
	cmd := &cobra.Command{
		Use:     "install-resource-group",
		Short:   livedocs.InstallResourceGroupShort,
		Long:    livedocs.InstallResourceGroupShort + "\n" + livedocs.InstallResourceGroupLong,
		Example: livedocs.InstallResourceGroupExamples,
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.Run(ioStreams.In, args)
		},
	}

	r.Command = cmd
	return r
}

// NewCmdInstallRG returns the cobra command for the install-resource-group command.
func NewCmdInstallRG(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	return GetInstallRGRunner(f, ioStreams).Command
}

// Run executes the installation of the ResourceGroup custom resource definition. Uses
// the current context of the kube config file (or the kube config flags) to
// determine the APIServer to install the CRD.
func (ir *InstallRGRunner) Run(reader io.Reader, args []string) error {
	// Validate the number of arguments.
	if len(args) > 0 {
		return fmt.Errorf("too many arguments; install-resource-group takes no arguments")
	}
	fmt.Fprint(ir.ioStreams.Out, "installing inventory ResourceGroup CRD...")
	err := live.InstallResourceGroupCRD(ir.factory)
	if err == nil {
		fmt.Fprintln(ir.ioStreams.Out, "success")
	} else {
		fmt.Fprintln(ir.ioStreams.Out, "failed")
	}
	return err
}
