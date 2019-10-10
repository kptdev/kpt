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

// Package cmdrc contains the rc command
package cmdrc

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/sets"
	"lib.kpt.dev/yaml"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "rc DIR...",
		Short: "Count Resources Config from a local package",
		Long: `Count Resources Config from a local package.

  DIR:
    Path to local package directory.
`,
		Example: `# print Resource counts from a package
kpt rc my-package/
`,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")
	c.Flags().BoolVar(&r.Kind, "kind", true,
		"count resources by kind.")

	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages bool
	Kind               bool
	C                  *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var inputs []kio.Reader
	for _, a := range args {
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath:        a,
			IncludeSubpackages: r.IncludeSubpackages,
		})
	}
	if len(inputs) == 0 {
		inputs = append(inputs, &kio.ByteReader{Reader: c.InOrStdin()})
	}

	var out []kio.Writer
	if r.Kind {
		out = append(out, kio.WriterFunc(func(nodes []*yaml.RNode) error {
			count := map[string]int{}
			k := sets.String{}
			for _, n := range nodes {
				m, _ := n.GetMeta()
				count[m.Kind]++
				k.Insert(m.Kind)
			}
			order := k.List()
			sort.Strings(order)
			for _, k := range order {
				fmt.Fprintf(c.OutOrStdout(), "%s: %d\n", k, count[k])
			}

			return nil
		}))

	} else {
		out = append(out, kio.WriterFunc(func(nodes []*yaml.RNode) error {
			fmt.Fprintf(c.OutOrStdout(), "%d\n", len(nodes))
			return nil
		}))
	}
	return kio.Pipeline{
		Inputs:  inputs,
		Outputs: out,
	}.Execute()
}
