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
	"github.com/GoogleContainerTools/kpt/internal/cmddestroy"
	"github.com/GoogleContainerTools/kpt/internal/cmdliveinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdmigrate"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/internal/util/cfgflags"
	"github.com/GoogleContainerTools/kpt/pkg/live"
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

	rgProvider := live.NewResourceGroupProvider(f)
	rgLoader := live.NewResourceGroupManifestLoader(f)

	cmProvider := provider.NewProvider(f)
	cmLoader := manifestreader.NewManifestLoader(f)

	// Init command which updates a Kptfile for the ResourceGroup inventory object.
	klog.V(2).Infoln("init command updates Kptfile for ResourceGroup inventory")
	initCmd := cmdliveinit.NewCommand(ctx, f, ioStreams)
	applyCmd := cmdapply.NewCommand(ctx, rgProvider, rgLoader, ioStreams)
	destroyCmd := cmddestroy.NewCommand(ctx, rgProvider, rgLoader, ioStreams)
	statusCmd := status.NewCommand(ctx, rgProvider, rgLoader, ioStreams)

	liveCmd.AddCommand(initCmd, applyCmd, destroyCmd, statusCmd)

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
