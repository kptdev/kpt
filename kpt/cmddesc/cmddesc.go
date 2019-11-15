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
	"kpt.dev/kpt/generated"
	"kpt.dev/kpt/util/cmdutil"
	"kpt.dev/kpt/util/desc"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "desc [DIR]...",
		Short:   generated.DescShort,
		Long:    generated.DescLong,
		Example: generated.DescExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}
	r.Command = c
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function
type Runner struct {
	Description desc.Command
	Command     *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		r.Description.PkgPaths = append(r.Description.PkgPaths, dir)
	}
	r.Description.StdOut = c.OutOrStdout()
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	r.Description.PkgPaths = append(r.Description.PkgPaths, args...)
	return r.Description.Run()
}
