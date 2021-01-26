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
	"os"

	"github.com/GoogleContainerTools/kpt/internal/cmdfetchk8sschema"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/cmd/destroy"
	"sigs.k8s.io/cli-utils/cmd/diff"
	"sigs.k8s.io/cli-utils/cmd/initcmd"
	"sigs.k8s.io/cli-utils/cmd/status"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

// resourceGroupEnv is an environment variable which hides code to implement
// the ResourceGroup inventory object and migrating to ResourceGroup inventory.
const resourceGroupEnv = "RESOURCE_GROUP_INVENTORY"

func GetLiveCommand(name string, f util.Factory) *cobra.Command {
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

	// The default provider is for ConfigMap inventory, but if the magic env
	// var exists, then the provider which handles both ConfigMap and ResourceGroup
	// inventory objects is used. If a package has both inventory objects, then
	// an error is thrown.
	var p provider.Provider = provider.NewProvider(f)
	var l manifestreader.ManifestLoader = manifestreader.NewManifestLoader(f)
	if _, exists := os.LookupEnv(resourceGroupEnv); exists {
		klog.V(2).Infoln("provider supports ResourceGroup and ConfigMap inventory")
		p = live.NewDualDelegatingProvider(f)
		l = live.NewDualDelegatingManifestReader(f)
	}

	// The default init command creates the ConfigMap inventory yaml. If the magic
	// env var exists, then we use the init command which updates a Kptfile for
	// the ResourceGroup inventory object.
	initCmd := initcmd.NewCmdInit(f, ioStreams)
	if _, exists := os.LookupEnv(resourceGroupEnv); exists {
		klog.V(2).Infoln("init command updates Kptfile for ResourceGroup inventory")
		initCmd = NewCmdInit(f, ioStreams)
	}
	initCmd.Short = livedocs.InitShort
	initCmd.Long = livedocs.InitShort + "\n" + livedocs.InitLong
	initCmd.Example = livedocs.InitExamples

	applyCmd := GetApplyRunner(p, l, ioStreams).Command()
	_ = applyCmd.Flags().MarkHidden("no-prune")
	applyCmd.Short = livedocs.ApplyShort
	applyCmd.Long = livedocs.ApplyShort + "\n" + livedocs.ApplyLong
	applyCmd.Example = livedocs.ApplyExamples

	previewCmd := GetPreviewRunner(p, l, ioStreams).Command()
	previewCmd.Short = livedocs.PreviewShort
	previewCmd.Long = livedocs.PreviewShort + "\n" + livedocs.PreviewLong
	previewCmd.Example = livedocs.PreviewExamples

	diffCmd := diff.NewCmdDiff(f, ioStreams)
	diffCmd.Short = livedocs.DiffShort
	diffCmd.Long = livedocs.DiffShort + "\n" + livedocs.DiffLong
	diffCmd.Example = livedocs.DiffExamples

	destroyCmd := destroy.GetDestroyRunner(p, l, ioStreams).Command
	destroyCmd.Short = livedocs.DestroyShort
	destroyCmd.Long = livedocs.DestroyShort + "\n" + livedocs.DestroyLong
	destroyCmd.Example = livedocs.DestroyExamples

	statusCmd := status.GetStatusRunner(p, l).Command
	statusCmd.Short = livedocs.StatusShort
	statusCmd.Long = livedocs.StatusLong
	statusCmd.Example = livedocs.StatusExamples

	fetchOpenAPICmd := cmdfetchk8sschema.NewCommand(name, f, ioStreams)

	liveCmd.AddCommand(initCmd, applyCmd, previewCmd, diffCmd, destroyCmd,
		fetchOpenAPICmd, statusCmd)

	// If the magic env var exists, then add the migrate to change
	// from ConfigMap to ResourceGroup inventory object. Also add
	// the install-resource-group command.
	if _, exists := os.LookupEnv(resourceGroupEnv); exists {
		klog.V(2).Infoln("adding kpt live migrate command")
		// Create a ConfigMap and a ResourceGroup provider for the
		// migrate command, and add the migrate command to live command.
		cmProvider := provider.NewProvider(f)
		rgProvider := live.NewResourceGroupProvider(f)
		cmLoader := manifestreader.NewManifestLoader(f)
		rgLoader := live.NewResourceGroupManifestLoader(f)
		migrateCmd := GetMigrateRunner(cmProvider, rgProvider, cmLoader, rgLoader, ioStreams).Command
		installRGCmd := GetInstallRGRunner(f, ioStreams).Command
		liveCmd.AddCommand(migrateCmd, installRGCmd)
	}

	return liveCmd
}
