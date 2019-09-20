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
	"kpt.dev/cmdtree"
	"kpt.dev/cmdtutorials"
	"kpt.dev/cmdupdate"
	"lib.kpt.dev/custom"
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
	cmd.AddCommand(cmdcat.Cmd().C)
	cmd.AddCommand(cmddesc.Cmd().C)
	cmd.AddCommand(cmdgrep.Cmd().C)
	cmd.AddCommand(cmddiff.Cmd().C)
	cmd.AddCommand(cmdfmt.Cmd().C)
	cmd.AddCommand(cmdget.Cmd().C)
	cmd.AddCommand(cmdbless.Cmd().C)
	cmd.AddCommand(cmdman.Cmd().C)
	cmd.AddCommand(cmdtree.Cmd().C)
	cmd.AddCommand(cmdupdate.Cmd().C)

	// help and documentation
	cmd.InitDefaultHelpCmd()
	cmd.AddCommand(cmdhelp.Kptfile)
	cmd.AddCommand(cmdhelp.PackageStructure)
	cmd.AddCommand(cmdtutorials.Tutorials)

	if len(os.Args) > 1 {
		if f, err := os.Stat(os.Args[1]); err == nil && f.IsDir() {
			os.Args[1] = strings.TrimSuffix(os.Args[1], "/")
			os.Args[1] = strings.TrimPrefix(os.Args[1], "./")
			name := filepath.Base(os.Args[1])
			err := custom.CommandBuilder{
				PkgPath: os.Args[1],
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
