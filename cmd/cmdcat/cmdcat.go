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

// Package cmdcat contains the fmt command
package cmdcat

import (
	"lib.kpt.dev/kio"

	"lib.kpt.dev/fmtr"

	"github.com/spf13/cobra"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "cat DIR...",
		Short: "Print Resource Config from a local package",
		Long: `Print Resource Config from a local package.

  DIR:
    Path to local package directory.
`,
		Example: `# print Resource config from a package
kpt cat my-package/
`,
		RunE:         r.runE,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")
	c.Flags().BoolVar(&r.Format, "format", true,
		"format resource config yaml before printing.")
	c.Flags().BoolVar(&r.KeepAnnotations, "annotate", true,
		"annotate resources with their file origins.")

	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages bool
	Format             bool
	KeepAnnotations    bool
	C                  *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var inputs []kio.Reader
	for _, a := range args {
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath: a,
		})
	}
	var filters []kio.Filter
	if r.Format {
		filters = append(filters, fmtr.Formatter{})
	}
	return kio.Pipeline{
		Inputs:  inputs,
		Filters: filters,
		Outputs: []kio.Writer{
			kio.ByteWriter{Writer: c.OutOrStdout()}},
	}.Execute()
}
