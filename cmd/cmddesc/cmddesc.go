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

// Package cmddesc contains the desc command
package cmddesc

import (
	"os"

	"github.com/spf13/cobra"
	"kpt.dev/internal/desc"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "desc [DIR]...",
		Short: "Display package description",
		Long: `Display package description.

Desc reads package information in given DIRs and displays it in tabular format.
Input can be a list of package directories (defaults to the current directory if not specifed).
Directory with a Kptfile is considered to be a valid package.
`,
		Example: `	# display description for package in current directory
	kpt desc

	# display description for packages in directories with 'prod-' prefix
	kpt desc prod-*
`,
		PreRunE:      r.preRunE,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	desc.Command
	C *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		r.PkgPaths = append(r.PkgPaths, dir)
	}
	r.Command.StdOut = c.OutOrStdout()
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	r.PkgPaths = append(r.PkgPaths, args...)
	return r.Run()
}
