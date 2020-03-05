// Copyright 2020 Google LLC
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

package commands

import (
	"flag"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/apply"
	"sigs.k8s.io/cli-utils/cmd/destroy"
	"sigs.k8s.io/cli-utils/cmd/initcmd"
	"sigs.k8s.io/cli-utils/cmd/preview"
)

func GetLiveCommand(name string) *cobra.Command {
	liveCmd := &cobra.Command{
		Use:   "live",
		Short: livedocs.READMEShort,
		Long:  livedocs.READMEShort + "\n" + livedocs.READMELong,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}

	// Create the factory and IOStreams for the "live" commands. The factory
	// is created using the config flags.
	flags := liveCmd.Flags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	matchVersionKubeConfigFlags := util.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionKubeConfigFlags.AddFlags(liveCmd.PersistentFlags())
	liveCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	f := util.NewFactory(matchVersionKubeConfigFlags)
	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	initCmd := initcmd.NewCmdInit(ioStreams)
	initCmd.Short = livedocs.InitShort
	initCmd.Long = livedocs.InitShort + "\n" + livedocs.InitLong

	applyCmd := apply.NewCmdApply(f, ioStreams)
	_ = applyCmd.Flags().MarkHidden("no-prune")
	applyCmd.Short = livedocs.ApplyShort
	applyCmd.Long = livedocs.ApplyShort + "\n" + livedocs.ApplyLong

	previewCmd := preview.NewCmdPreview(f, ioStreams)
	previewCmd.Short = livedocs.PreviewShort
	previewCmd.Long = livedocs.PreviewShort + "\n" + livedocs.PreviewLong

	destroyCmd := destroy.NewCmdDestroy(f, ioStreams)
	destroyCmd.Short = livedocs.DestroyShort
	destroyCmd.Long = livedocs.DestroyShort + "\n" + livedocs.DestroyLong

	liveCmd.AddCommand(initCmd, applyCmd, previewCmd, destroyCmd)

	return liveCmd
}
