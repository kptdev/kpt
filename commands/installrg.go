// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"fmt"
	"io"

	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
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
		Use:                   "install-resource-group",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Install ResourceGroup custom resource definition as inventory object into APIServer"),
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
	fmt.Fprint(ir.ioStreams.Out, "installing ResourceGroup custom resource definition...")
	// Apply the ResourceGroup CRD to the cluster, swallowing an "AlreadyExists" error.
	err := live.ApplyResourceGroupCRD(ir.factory)
	if apierrors.IsAlreadyExists(err) {
		fmt.Fprint(ir.ioStreams.Out, "already installed...")
		err = nil
	}
	if err != nil {
		fmt.Fprintln(ir.ioStreams.Out, "failed")
	} else {
		fmt.Fprintln(ir.ioStreams.Out, "success")
	}
	return err
}
