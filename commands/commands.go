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
	"github.com/GoogleContainerTools/kpt/internal/cmdcomplete"
	"github.com/GoogleContainerTools/kpt/internal/cmddesc"
	"github.com/GoogleContainerTools/kpt/internal/cmddiff"
	"github.com/GoogleContainerTools/kpt/internal/cmdget"
	"github.com/GoogleContainerTools/kpt/internal/cmdinit"
	"github.com/GoogleContainerTools/kpt/internal/cmdman"
	"github.com/GoogleContainerTools/kpt/internal/cmdsub"
	"github.com/GoogleContainerTools/kpt/internal/cmdsync"
	"github.com/GoogleContainerTools/kpt/internal/cmdtutorials"
	"github.com/GoogleContainerTools/kpt/internal/cmdupdate"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/posener/complete/v2"
	"github.com/spf13/cobra"
)

// GetAllCommands returns the set of kpt commands to be registered
func GetAllCommands(name string) []*cobra.Command {
	c := []*cobra.Command{
		cmdcomplete.NewCommand(name),
		cmddesc.NewCommand(name),
		cmdget.NewCommand(name),
		cmdinit.NewCommand(name),
		cmdman.NewCommand(name),
		cmdsync.NewCommand(name),
		cmdsub.NewCommand(name),
		cmdupdate.NewCommand(name),
		cmddiff.NewCommand(name),
	}
	c = append(c, cmdtutorials.Tutorials(name)...)

	// apply cross-cutting issues to commands
	for i := range c {
		cmd := c[i]
		// check if silencing errors is off
		cmdutil.SetSilenceErrors(cmd)

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
	}
	return c
}

var allCommands = map[string]func(string) *cobra.Command{
	"desc":               cmddesc.NewCommand,
	"diff":               cmddiff.NewCommand,
	"get":                cmdget.NewCommand,
	"init":               cmdinit.NewCommand,
	"man":                cmdman.NewCommand,
	"sub":                cmdsub.NewCommand,
	"sync":               cmdsync.NewCommand,
	"update":             cmdupdate.NewCommand,
	"install-completion": cmdcomplete.NewCommand,
}

func GetCommands(name string, commands ...string) []*cobra.Command {
	var c []*cobra.Command
	for i := range commands {
		c = append(c, allCommands[commands[i]](name))
	}
	return c
}

// Complete returns a completion command
func Complete(cmd *cobra.Command, fn cmdcomplete.VisitFlags) *complete.Command {
	return cmdcomplete.Complete(cmd, fn)
}
