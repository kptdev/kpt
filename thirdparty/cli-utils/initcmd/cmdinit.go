// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package initcmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/cli-utils/pkg/config"
)

// InitRunner encapsulates the structures for the init command.
type InitRunner struct {
	Command     *cobra.Command
	InitOptions *config.InitOptions
}

// GetInitRunner builds and returns the InitRunner. Connects the InitOptions.Run
// to the cobra command.
func GetInitRunner(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *InitRunner {
	io := config.NewInitOptions(f, ioStreams)
	cmd := &cobra.Command{
		Use:                   "init DIRECTORY",
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Create a prune manifest ConfigMap as a inventory object"),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := io.Complete(args)
			if err != nil {
				return err
			}
			return io.Run()
		},
	}
	cmd.Flags().StringVarP(&io.InventoryID, "inventory-id", "i", "", "Identifier for group of applied resources. Must be composed of valid label characters.")
	i := &InitRunner{
		Command:     cmd,
		InitOptions: io,
	}
	return i
}

// NewCmdInit returns the cobra command for the init command.
func NewCmdInit(f cmdutil.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	return GetInitRunner(f, ioStreams).Command
}
