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

// Package cndcat contains the fmt command
package cmdtree

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "tree DIR",
		Short: "Display package Resource structure",
		Long: `Display package Resource structure.

  DIR:
    Path to local package directory.
`,
		Example: `# print package structure
kpt tree my-package/
`,
		RunE:         r.runE,
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")

	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages bool
	C                  *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var input kio.Reader
	var root = "."
	if len(args) == 1 {
		root = filepath.Clean(args[0])
		input = kio.LocalPackageReader{
			PackagePath: args[0],
		}
	} else {
		input = kio.ByteReader{
			Reader: c.InOrStdin(),
		}
	}
	return kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Outputs: []kio.Writer{kio.TreeWriter{Root: root, Writer: c.OutOrStdout()}},
	}.Execute()
}
