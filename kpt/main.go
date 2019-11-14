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

//go:generate go run ./mdtogo ../docs/tutorials ./cmdtutorials/generated --full=true
//go:generate go run ./mdtogo ../docs/commands ./generated
package main

import (
	"os"

	"github.com/spf13/cobra"
	"kpt.dev/kpt/cmdbless"
	"kpt.dev/kpt/cmddesc"
	"kpt.dev/kpt/cmdget"
	"kpt.dev/kpt/cmdman"
	"kpt.dev/kpt/cmdtutorials"
	"kpt.dev/kpt/cmdupdate"
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
	cmd.AddCommand(cmddesc.NewCommand())
	cmd.AddCommand(cmdget.NewCommand())
	cmd.AddCommand(cmdbless.NewCommand())
	cmd.AddCommand(cmdman.NewCommand())
	cmd.AddCommand(cmdupdate.NewCommand())

	// help and documentation
	cmd.InitDefaultHelpCmd()
	for i := range cmdtutorials.Tutorials {
		cmd.AddCommand(cmdtutorials.Tutorials[i])
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
