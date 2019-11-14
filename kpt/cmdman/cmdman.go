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

// Package cmdman contains the man command.
package cmdman

import (
	"github.com/spf13/cobra"
	"kpt.dev/kpt/util/man"
)

// NewRunner returns a command runner.
func NewRunner() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "man LOCAL_PKG_DIR",
		Short: "Format and display package documentation if it exists",
		Long: `Format and display package documentation if it exists.
Args:

  LOCAL_PKG_DIR:
    path to locally fetched package.

  If package documentation is missing or 'man' is not installed, the command will fail.`,
		Example: `  # display package documentation
  kpt man my-package/

  # display subpackage documentation
  kpt man my-package/sub-package/`, RunE: r.runE,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		PreRunE:      r.preRunE,
		SuggestFor:   []string{"docs"},
	}

	r.Command = c
	return r
}

func NewCommand() *cobra.Command {
	return NewRunner().Command
}

type Runner struct {
	Man     man.Command
	Command *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.Man.Path = "."
	if len(args) > 0 {
		r.Man.Path = args[0]
	}
	r.Man.StdOut = c.OutOrStdout()
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return r.Man.Run()
}
