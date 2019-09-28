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
		Use:   "merge",
		Short: "Merge Resource configuration files",
		Long: `Merge Resource configuration files

Merge reads Kubernetes Resource yaml configuration files from stdin and write the result to
stdout.

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
		Args:         cobra.ExactArgs(0),
	}
	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	C *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return kio.Pipeline{
		Inputs:  []kio.Reader{kio.ByteReader{Reader: c.InOrStdin()}},
		Filters: []kio.Filter{filters.MergeFilter{}},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: c.OutOrStdout()}},
	}.Execute()
}
