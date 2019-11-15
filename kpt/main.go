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

const name = "kpt"

var cmd = &cobra.Command{
	Use:   name,
	Short: "K Packaging Tool",
	Long: `Description:
  Build, compose and customize packages of configuration.
	
  For best results, use with tools such as kustomize and kubectl.`,
	Example: name + ` help tutorials`,
}

func main() {
	// sorted lexicographically
	cmd.AddCommand(cmddesc.NewCommand(name))
	cmd.AddCommand(cmdget.NewCommand(name))
	cmd.AddCommand(cmdbless.NewCommand(name))
	cmd.AddCommand(cmdman.NewCommand(name))
	cmd.AddCommand(cmdupdate.NewCommand(name))

	// help and documentation
	cmd.InitDefaultHelpCmd()
	tutorials := cmdtutorials.Tutorials(name)
	for i := range tutorials {
		cmd.AddCommand(tutorials[i])
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
