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

//go:generate go run ./ gen-docs ../docs/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"kpt.dev/cmdbless"
	"kpt.dev/cmdcat"
	"kpt.dev/cmddesc"
	"kpt.dev/cmddiff"
	"kpt.dev/cmdfmt"
	"kpt.dev/cmdgendocs"
	"kpt.dev/cmdget"
	"kpt.dev/cmdgrep"
	"kpt.dev/cmdhelp"
	"kpt.dev/cmdman"
	"kpt.dev/cmdmerge"
	"kpt.dev/cmdrc"
	"kpt.dev/cmdreconcile"
	"kpt.dev/cmdtree"
	"kpt.dev/cmdtutorials"
	"kpt.dev/cmdupdate"
	"lib.kpt.dev/command"
)

var cmd = &cobra.Command{
	Use:   "kpt",
	Short: "Kpt Packaging Tool",
	Long: `Description:
  Build, compose and customize Kubernetes Resource packages.
	
  For best results, use with tools such as kustomize and kubectl.`,
	Example: ` 
  kpt help tutorials`,
}

func main() {
	// sorted lexicographically
	cmd.AddCommand(cmdcat.Cmd().CobraCommand)
	cmd.AddCommand(cmddesc.Cmd().C)
	cmd.AddCommand(cmdgrep.Cmd().C)
	cmd.AddCommand(cmddiff.Cmd().C)
	cmd.AddCommand(cmdfmt.Cmd().C)
	cmd.AddCommand(cmdget.Cmd().C)
	cmd.AddCommand(cmdbless.Cmd().C)
	cmd.AddCommand(cmdman.Cmd().C)
	cmd.AddCommand(cmdmerge.Cmd().C)
	cmd.AddCommand(cmdrc.Cmd().C)
	cmd.AddCommand(cmdreconcile.Cmd().C)
	cmd.AddCommand(cmdtree.Cmd().C)
	cmd.AddCommand(cmdupdate.Cmd().C)

	// help and documentation
	cmd.InitDefaultHelpCmd()
	cmd.AddCommand(cmdhelp.Apis)
	cmd.AddCommand(cmdtutorials.Tutorials)
	cmd.AddCommand(command.HelpCommand)

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "help" {
			arg = os.Args[2]
		}

		if f, err := os.Stat(arg); (err == nil && f.IsDir()) || command.IsWildcardPath(arg) {
			arg = strings.TrimSuffix(arg, "/")
			arg = strings.TrimPrefix(arg, "./")
			if os.Args[1] == "help" {
				os.Args[2] = arg
			} else {
				os.Args[1] = arg
			}
			name := filepath.Base(arg)

			cmd.AddCommand(&cobra.Command{
				Use:   arg,
				Short: fmt.Sprintf("%s package specific commands", arg),
				Long: fmt.Sprintf(`%s package specific commands.

Contains commands enabled specifically for the %s package -- either through duck-typing off
the structure of the Resources, or through custom commands published as part of the package
itself.`, arg, arg),
			})

			// Duck commands
			if err := command.AddCommands(arg, cmd); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}

			// Custom commands
			err := command.CommandBuilder{
				PkgPath: arg,
				RootCmd: cmd,
				Name:    name,
				CmdPath: []string{name},
			}.BuildCommands()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		}
	}

	cmd.AddCommand(cmdgendocs.Cmd(cmd))

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
