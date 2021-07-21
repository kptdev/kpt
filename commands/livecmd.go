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
	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/status"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
	cluster "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/util/factory"
)

func GetLiveCommand(ctx context.Context, _, version string) *cobra.Command {
	liveCmd := &cobra.Command{
		Use:   "live",
		Short: livedocs.LiveShort,
		Long:  livedocs.LiveShort + "\n" + livedocs.LiveLong,
	}

	ioStreams := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	f := newFactory(liveCmd, version)

	// Init command which updates a Kptfile for the ResourceGroup inventory object.
	klog.V(2).Infoln("init command updates Kptfile for ResourceGroup inventory")
	initCmd := cmdliveinit.NewCommand(ctx, f, ioStreams)
	applyCmd := cmdapply.NewCommand(ctx, f, ioStreams)
	destroyCmd := cmddestroy.NewCommand(ctx, f, ioStreams)
	statusCmd := status.NewCommand(ctx, f)
	installRGCmd := GetInstallRGRunner(f, ioStreams).Command
	liveCmd.AddCommand(initCmd, applyCmd, destroyCmd, statusCmd, installRGCmd)

	// Add the migrate command to change from ConfigMap to ResourceGroup inventory
	// object.
	klog.V(2).Infoln("adding kpt live migrate command")
	// TODO: Remove the loader implementation for ConfigMap once we remove the
	// migrate command.
	cmLoader := manifestreader.NewManifestLoader(f)
	migrateCmd := cmdmigrate.NewCommand(ctx, f, cmLoader, ioStreams)

	liveCmd.AddCommand(migrateCmd)

	return liveCmd
}

func newFactory(cmd *cobra.Command, version string) cluster.Factory {
	flags := cmd.PersistentFlags()
	kubeConfigFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	kubeConfigFlags.AddFlags(flags)
	// Use the CachingRESTClientGetter to make sure the same RESTMapper is shared
	// across all kpt live functionality. This is needed to properly invalidate the
	// RESTMapper cache when CRDs have been installed in the cluster.
	cachingConfigFlags := &factory.CachingRESTClientGetter{
		Delegate: kubeConfigFlags,
	}
	userAgentKubeConfigFlags := &cfgflags.UserAgentKubeConfigFlags{
		Delegate:  cachingConfigFlags,
		UserAgent: fmt.Sprintf("kpt/%s", version),
	}
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cluster.NewFactory(userAgentKubeConfigFlags)
}
