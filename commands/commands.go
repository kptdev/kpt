// Copyright 2019 Google LLC
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
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/cmddesc"
	"github.com/GoogleContainerTools/kpt/internal/cmddiff"
	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdman"
	"github.com/GoogleContainerTools/kpt/internal/cmdsync"
	"github.com/GoogleContainerTools/kpt/internal/cmdtutorials"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	configdocs "github.com/GoogleContainerTools/kpt/internal/docs/generated/config"
	pkgdocs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/cmd/config/configcobra"
	"sigs.k8s.io/kustomize/cmd/kubectl/kubectlcobra"
	"sigs.k8s.io/kustomize/cmd/resource/status"
)

func GetAnthosCommands(name string) []*cobra.Command {
	c := append([]*cobra.Command{cmddesc.NewCommand(name),
		cmdget.NewCommand(name), cmdinit.NewCommand(name),
		cmdman.NewCommand(name), cmdsync.NewCommand(name),
		cmdupdate.NewCommand(name), cmddiff.NewCommand(name),
	}, cmdtutorials.Tutorials(name)...)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}

// NormalizeCommand will modify commands to be consistent, e.g. silencing errors
func NormalizeCommand(c ...*cobra.Command) {
	for i := range c {
		cmd := c[i]
		// check if silencing errors is off
		cmdutil.SetSilenceErrors(cmd)
		cmd.Short = strings.TrimPrefix(cmd.Short, "[Alpha] ")

		// check if stack printing is on
		if cmd.PreRunE != nil {
			fn := cmd.PreRunE
			cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
				err := fn(cmd, args)
				return cmdutil.HandlePreRunError(cmd, err)
			}
		}
		if cmd.RunE != nil {
			fn := cmd.RunE
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				err := fn(cmd, args)
				return cmdutil.HandleError(cmd, err)
			}
		}
		NormalizeCommand(cmd.Commands()...)
	}
}

// GetKptCommands returns the set of kpt commands to be registered
func GetKptCommands(name string) []*cobra.Command {
	var c []*cobra.Command
	cfgCmd := &cobra.Command{
		Use:     "config",
		Short:   configdocs.READMEShort,
		Long:    configdocs.READMEShort + "\n" + configdocs.READMELong,
		Example: configdocs.READMEExamples,
	}

	cat := configcobra.Cat(name)
	count := configcobra.Count(name)
	createSetter := configcobra.CreateSetter(name)
	fmt := configcobra.Fmt(name)
	grep := configcobra.Grep(name)
	listSetters := configcobra.ListSetters(name)
	merge := configcobra.Merge(name)
	merge3 := configcobra.Merge3(name)
	set := configcobra.Set(name)
	tree := configcobra.Tree(name)
	cfgCmd.AddCommand(cat, count, createSetter, fmt, grep, listSetters, merge, merge3, set, tree)

	request := &cobra.Command{
		Use:   "http",
		Short: "Apply and make Resource requests to clusters",
	}
	request.AddCommand(status.StatusCommand())
	request.AddCommand(kubectlcobra.GetCommand(nil).Commands()...)

	pkg := &cobra.Command{
		Use:     "pkg",
		Short:   pkgdocs.READMEShort,
		Long:    pkgdocs.READMELong,
		Example: pkgdocs.READMEExamples,
	}
	pkg.AddCommand(cmddesc.NewCommand(name), cmdget.NewCommand(name), cmdinit.NewCommand(name),
		cmdman.NewCommand(name), cmdsync.NewCommand(name), cmdupdate.NewCommand(name),
		cmddiff.NewCommand(name))

	functions := &cobra.Command{
		Use:   "functions",
		Short: "Generate and mutate local configuration by running functional images",
	}
	var remove []*cobra.Command
	for i := range cfgCmd.Commands() {
		c := cfgCmd.Commands()[i]
		if strings.HasPrefix(cfgCmd.Commands()[i].Use, "run") {
			functions.AddCommand(c)
			remove = append(remove, c)
			continue
		}
		if strings.HasPrefix(cfgCmd.Commands()[i].Use, "source") {
			functions.AddCommand(c)
			remove = append(remove, c)
			continue
		}
		if strings.HasPrefix(cfgCmd.Commands()[i].Use, "sink") {
			functions.AddCommand(c)
			remove = append(remove, c)
			continue
		}
	}
	for i := range remove {
		cfgCmd.RemoveCommand(remove[i])
	}

	tutorials := &cobra.Command{
		Use:   "tutorials",
		Short: "Tutorials for using kpt",
	}

	tutorials.AddCommand(cmdtutorials.Tutorials(name)...)
	c = append(c, request, cfgCmd, pkg, tutorials, functions)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}
