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

// Package cmdmerge contains the merge command
package cmdmerge

import (
	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "merge [SOURCE_DIR...] [DESTINATION_DIR]",
		Short: "Merge Resource configuration files",
		Long: `Merge Resource configuration files

Merge reads Kubernetes Resource yaml configuration files from stdin or sources packages and write
the result to stdout or a destination package.

Resources are merged using the Resource [apiVersion, kind, name, namespace] as the key.  If any of
these are missing, merge will default the missing values to empty.

Resources specified later are high-precedence (the source) and Resources specified
earlier are lower-precedence (the destination).

Merge uses the following rules for merging a source Resource into a destination Resource:

- Map fields specified in both the source and destination are merged recursively.
- Scalar fields specified in both the source and destination have the destination value replaced
  by the source value.
- Lists elements specified in both the source and destination are merged:
  - As a scalar if the list elements do not have an associative key.
  - As maps if the lists do have an associative key -- the associative key is used as the map key
  - The following are associative in precedence order:
    "mountPath", "devicePath", "ip", "type", "topologyKey", "name", "containerPort"
- Any fields specified only in the destination are kept in the destination as is.
- Any fields specified only in the source are copied to the destination.
- Fields specified in the sources as null will be cleared from the destination if present
- Comments are merged on all fields and list elements from the source if they are specified,
  on the source, otherwise the destination comments are kept as is.
`,
		Example:      `cat resources_and_patches.yaml | kpt merge > merged_resources.yaml`,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	r.C = c
	r.C.Flags().BoolVar(&r.InvertOrder, "invert-order", false,
		"if true, merge Resources in the reverse order")
	return r
}

// Runner contains the run function
type Runner struct {
	C           *cobra.Command
	InvertOrder bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	var inputs []kio.Reader
	// add the packages in reverse order -- the arg list should be highest precedence first
	// e.g. merge from -> to, but the MergeFilter is highest precedence last
	for i := len(args) - 1; i >= 0; i-- {
		inputs = append(inputs, kio.LocalPackageReader{PackagePath: args[i]})
	}
	// if there is no "from" package, read from stdin
	rw := &kio.ByteReadWriter{
		Reader:                c.InOrStdin(),
		Writer:                c.OutOrStdout(),
		KeepReaderAnnotations: true,
	}
	if len(inputs) < 2 {
		inputs = append(inputs, rw)
	}

	// write to the "to" package if specified
	var outputs []kio.Writer
	if len(args) != 0 {
		outputs = append(outputs, kio.LocalPackageWriter{PackagePath: args[len(args)-1]})
	}
	// if there is no "to" package, write to stdout
	if len(outputs) == 0 {
		outputs = append(outputs, rw)
	}

	filters := []kio.Filter{filters.MergeFilter{}, filters.FormatFilter{}}
	return kio.Pipeline{Inputs: inputs, Filters: filters, Outputs: outputs}.Execute()
}
