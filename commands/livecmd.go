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
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/cmdapply"
	"github.com/GoogleContainerTools/kpt/internal/cmdliveinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdmigrate"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/util/cfgflags"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/destroy"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/diff"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/preview"
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	cluster "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
)

func GetLiveCommand(ctx context.Context, _, version string) *cobra.Command {
	liveCmd := &cobra.Command{
		Use:   "live",
		Short: livedocs.LiveShort,
		Long:  livedocs.LiveShort + "\n" + livedocs.LiveLong,
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

	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	f := newFactory(liveCmd, version)

	// The provider handles both ConfigMap and ResourceGroup inventory objects.
	// If a package has both inventory objects, then an error is thrown.
	klog.V(2).Infoln("provider supports ResourceGroup and ConfigMap inventory")
	dp := live.NewDualDelegatingProvider(f)
	dl := live.NewDualDelegatingManifestReader(f)

	rgProvider := live.NewResourceGroupProvider(f)
	rgLoader := live.NewResourceGroupManifestLoader(f)

	cmProvider := provider.NewProvider(f)
	cmLoader := manifestreader.NewManifestLoader(f)

	// Init command which updates a Kptfile for the ResourceGroup inventory object.
	klog.V(2).Infoln("init command updates Kptfile for ResourceGroup inventory")
	initCmd := cmdliveinit.NewCommand(ctx, f, ioStreams)

	applyCmd := cmdapply.NewCommand(ctx, rgProvider, rgLoader, ioStreams)

	previewCmd := preview.GetPreviewRunner(dp, dl, ioStreams).Command
	previewCmd.Short = livedocs.PreviewShort
	previewCmd.Long = livedocs.PreviewShort + "\n" + livedocs.PreviewLong
	previewCmd.Example = livedocs.PreviewExamples

	diffCmd := diff.NewCmdDiff(f, ioStreams)
	diffCmd.Short = livedocs.DiffShort
	diffCmd.Long = livedocs.DiffShort + "\n" + livedocs.DiffLong
	diffCmd.Example = livedocs.DiffExamples

	destroyCmd := destroy.GetDestroyRunner(dp, dl, ioStreams).Command
	destroyCmd.Short = livedocs.DestroyShort
	destroyCmd.Long = livedocs.DestroyShort + "\n" + livedocs.DestroyLong
	destroyCmd.Example = livedocs.DestroyExamples

	statusCmd := status.GetStatusRunner(dp, dl).Command
	statusCmd.Short = livedocs.StatusShort
	statusCmd.Long = livedocs.StatusLong
	statusCmd.Example = livedocs.StatusExamples

	liveCmd.AddCommand(initCmd, applyCmd, previewCmd, diffCmd, destroyCmd,
		statusCmd)

	// Add the migrate command to change from ConfigMap to ResourceGroup inventory
	// object. Also add the install-resource-group command.
	klog.V(2).Infoln("adding kpt live migrate command")
	migrateCmd := cmdmigrate.NewCommand(ctx, cmProvider, rgProvider, cmLoader, rgLoader, ioStreams)
	installRGCmd := GetInstallRGRunner(f, ioStreams).Command
	liveCmd.AddCommand(migrateCmd, installRGCmd)

	return liveCmd
}

func newFactory(cmd *cobra.Command, version string) cluster.Factory {
	flags := cmd.PersistentFlags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	userAgentKubeConfigFlags := &cfgflags.UserAgentKubeConfigFlags{
		Delegate:  kubeConfigFlags,
		UserAgent: fmt.Sprintf("kpt/%s", version),
	}
	matchVersionKubeConfigFlags := cluster.NewMatchVersionFlags(
		&factory.CachingRESTClientGetter{
			Delegate: userAgentKubeConfigFlags,
		},
	)
	matchVersionKubeConfigFlags.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cluster.NewFactory(matchVersionKubeConfigFlags)
}
