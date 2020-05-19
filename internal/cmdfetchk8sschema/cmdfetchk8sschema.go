// Copyright 2020 Google LLC
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

package cmdfetchk8sschema

import (
	"encoding/json"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/openapi"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/util"
)

func NewRunner(parent string, f util.Factory,
	ioStreams genericclioptions.IOStreams) *Runner {
	r := &Runner{
		IOStreams: ioStreams,
		Factory:   f,
	}
	// TODO: Update description with info from the site.
	c := &cobra.Command{
		Use:   "fetch-k8s-schema",
		Short: "Fetch kubernetes schema from cluster and print to stdout",
		Long: `
Fetch kubernetes schema from cluster and print to stdout

  kpt live fetch-k8s-schema [flags]

Flags:
  --pretty-print:
    Format the schema before printing it
`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Don't run the PreRun functions that read k8s-schema since
			// we are doing that in the command itself.
			return nil
		},
		RunE: r.runE,
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c

	c.Flags().BoolVar(&r.PrettyPrint, "pretty-print", false,
		`Format the schema before printing`)

	return r
}

func NewCommand(parent string, f util.Factory,
	ioStreams genericclioptions.IOStreams) *cobra.Command {
	return NewRunner(parent, f, ioStreams).Command
}

// Runner contains the run function
type Runner struct {
	Command   *cobra.Command
	IOStreams genericclioptions.IOStreams
	Factory   util.Factory

	PrettyPrint bool
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	schema, err := openapi.FetchOpenAPISchemaFromCluster(r.Factory)
	if err != nil {
		return err
	}

	if r.PrettyPrint {
		var data map[string]interface{}
		err = json.Unmarshal(schema, &data)
		if err != nil {
			return err
		}

		schema, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(r.IOStreams.Out, "%s\n", string(schema))
	return nil
}
