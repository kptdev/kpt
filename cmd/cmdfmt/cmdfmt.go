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

// Package cmdfmt contains the fmt command
package cmdfmt

import (
	"lib.kpt.dev/kio"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio/filters"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "fmt",
		Short: "Format yaml configuration files",
		Long: `Format yaml configuration files

Fmt will format input by ordering fields and unordered list items in Kubernetes
objects.  Inputs may be directories, files or stdin, and their contents must
include both apiVersion and kind fields.

- Stdin inputs are formatted and written to stdout
- File inputs (args) are formatted and written back to the file
- Directory inputs (args) are walked, each encountered .yaml and .yml file
  acts as an input

For inputs which contain multiple yaml documents separated by \n---\n,
each document will be formatted and written back to the file in the original
order.

Field ordering roughly follows the ordering defined in the source Kubernetes
resource definitions (i.e. go structures), falling back on lexicographical
sorting for unrecognized fields.

Unordered list item ordering is defined for specific Resource types and
field paths.

- .spec.template.spec.containers (by element name)
- .webhooks.rules.operations (by element value)
`,
		Example: `
	# format file1.yaml and file2.yml
	kpt fmt file1.yaml file2.yml

	# format all *.yaml and *.yml recursively traversing directories
	kpt fmt dir/

	# format kubectl output
	kubectl get -o yaml deployments | kpt fmt

	# format kustomize output
	kustomize build | kpt fmt
`,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	c.Flags().StringVar(&r.FilenamePattern, "pattern", filters.DefaultFilenamePattern,
		`pattern to use for generating filenames for resources -- may contain the following
formatting substitution verbs {'%n': 'metadata.name', '%s': 'metadata.namespace', '%k': 'kind'}`)
	c.Flags().BoolVar(&r.SetFilenames, "set-filenames", false,
		`if true, set default filenames on Resources without them`)
	r.C = c
	return r
}

// Runner contains the run function
type Runner struct {
	C               *cobra.Command
	FilenamePattern string
	SetFilenames    bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	f := []kio.Filter{filters.FormatFilter{}}

	// format with file names
	if r.SetFilenames {
		f = append(f, &filters.FileSetter{FilenamePattern: r.FilenamePattern})
	}

	// format stdin if there are no args
	if len(args) == 0 {
		return kio.Pipeline{
			Inputs:  []kio.Reader{kio.ByteReader{Reader: c.InOrStdin()}},
			Filters: f,
			Outputs: []kio.Writer{kio.ByteWriter{Writer: c.OutOrStdout()}},
		}.Execute()
	}

	for i := range args {
		path := args[i]
		rw := kio.LocalPackageReadWriter{PackagePath: path}
		err := kio.Pipeline{
			Inputs: []kio.Reader{rw}, Filters: f, Outputs: []kio.Writer{rw}}.Execute()
		if err != nil {
			return err
		}
	}
	return nil
}
