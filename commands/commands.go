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
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
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
	cfgCmd := GetConfigCommand(name)
	httpCmd := GetSvrCommand(name)
	fnCmd := GetFnCommand(name)
	pkgCmd := GetPkgCommand(name)

	//tutorials := &cobra.Command{
	//	Use:   "tutorials",
	//	Short: "Tutorials for using kpt",
	//}
	//
	//tutorials.AddCommand(cmdtutorials.Tutorials(name)...)
	c = append(c, httpCmd, cfgCmd, pkgCmd, fnCmd)

	// apply cross-cutting issues to commands
	NormalizeCommand(c...)
	return c
}
