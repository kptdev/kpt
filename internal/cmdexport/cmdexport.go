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

// package cmdexport contains the export command.
package cmdexport

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/orchestrators"
	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/spf13/cobra"
)

// The `kpt fn export` command.
func ExportCommand() *cobra.Command {
	return GetExportRunner().Command
}

// GetExportRunner creates a ExportRunner instance and wires it to the corresponding Command.
func GetExportRunner() *ExportRunner {
	r := &ExportRunner{PipelineConfig: &types.PipelineConfig{}}
	c := &cobra.Command{
		Use:     "export orchestrator DIR/",
		Short:   fndocs.ExportShort,
		Long:    fndocs.ExportLong,
		Example: fndocs.ExportExamples,
		// Validate and parse args.
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("accepts %d args, received %d", 2, len(args))
			}

			r.Orchestrator, r.Dir = args[0], args[1]

			switch r.Orchestrator {
			case "github-actions":
				{
					r.Pipeline = new(orchestrators.GitHubActions)
				}
			default:
				return fmt.Errorf("unsupported orchestrator %v", r.Orchestrator)
			}

			return nil
		},
		RunE: r.runE,
	}

	c.Flags().StringSliceVar(
		&r.FnPaths, "fn-path", []string{},
		"read functions from these directories instead of the configuration directory.")
	c.Flags().StringVar(
		&r.OutputFilePath, "output", "",
		"specify the filename of the generated pipeline. If omitted, the default output is stdout")

	r.Command = c

	return r
}

// The ExportRunner wraps the user's input and runs the command.
type ExportRunner struct {
	Orchestrator   string
	OutputFilePath string
	Command        *cobra.Command
	*types.PipelineConfig
	Pipeline orchestrators.Pipeline
}

// runE generates the pipeline and writes it into a file or stdout.
func (r *ExportRunner) runE(c *cobra.Command, args []string) error {
	pipeline := r.Pipeline.Init(r.PipelineConfig).Generate()

	if r.OutputFilePath != "" {
		fo, err := os.Create(r.OutputFilePath)

		if err != nil {
			return err
		}

		c.SetOut(fo)
	}

	_, err := c.OutOrStdout().Write(pipeline)

	return err
}
