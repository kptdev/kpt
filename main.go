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

//go:generate $GOBIN/mdtogo docs/tutorials internal/docs/generated/tutorials --full=true --license=none
//go:generate $GOBIN/mdtogo docs/commands internal/docs/generated/commands --license=none
package main

import (
	"os"

	"github.com/GoogleContainerTools/kpt/commands"
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "kpt",
		Short:   docs.KptShort,
		Long:    docs.KptLong,
		Example: docs.KptExamples,
	}

	// help and documentation
	cmd.InitDefaultHelpCmd()
	cmd.AddCommand(commands.GetCommands("kpt")...)

	// enable stack traces
	cmd.PersistentFlags().BoolVar(&cmdutil.StackOnError, "stack-trace", false,
		"print a stack-trace on failure")

	// exit on an error
	cmdutil.ExitOnError = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
