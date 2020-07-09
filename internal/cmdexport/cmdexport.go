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
	"sort"
	"strings"

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
		Use:     "export DIR/",
		Short:   fndocs.ExportShort,
		Long:    fndocs.ExportLong,
		Example: fndocs.ExportExamples,
		Args:    cobra.ExactArgs(1),
		PreRunE: r.preRunE,
		RunE:    r.runE,
	}

	c.Flags().StringVarP(
		&r.WorkflowOrchestrator, "workflow", "w", "",
		fmt.Sprintf(
			"specify the workflow orchestrator that the pipeline is generated for. Supported workflow orchestrators are %s.",
			listSupportedOrchestrators()),
	)
	_ = c.MarkFlagRequired("workflow")
	c.Flags().StringSliceVar(
		&r.FnPaths, "fn-path", []string{},
		"read functions from these directories instead of the configuration directory.",
	)
	c.Flags().StringVar(
		&r.OutputFilePath, "output", "",
		"specify the filename of the generated pipeline. If omitted, the default output is stdout.",
	)

	r.Command = c

	return r
}

// The ExportRunner wraps the user's input and runs the command.
type ExportRunner struct {
	WorkflowOrchestrator string
	OutputFilePath       string
	Command              *cobra.Command
	*types.PipelineConfig
	Pipeline orchestrators.Pipeline
}

func (r *ExportRunner) preRunE(cmd *cobra.Command, args []string) (err error) {
	r.Dir = args[0]

	if len(r.WorkflowOrchestrator) == 0 {
		return fmt.Errorf(
			"--workflow flag is required. It must be one of %s",
			listSupportedOrchestrators(),
		)
	}

	r.Pipeline = supportedOrchestrators()[r.WorkflowOrchestrator]
	if r.Pipeline == nil {
		return fmt.Errorf(
			"unsupported orchestrator %v. It must be one of %s",
			r.WorkflowOrchestrator,
			listSupportedOrchestrators(),
		)
	}

	r.CWD, err = os.Getwd()
	if err != nil {
		return
	}

	err = r.CheckFnPaths()
	if err != nil {
		return
	}

	return r.PipelineConfig.UseRelativePaths()
}

// runE generates the pipeline and writes it into a file or stdout.
func (r *ExportRunner) runE(c *cobra.Command, args []string) error {
	pipeline, e := r.Pipeline.Init(r.PipelineConfig).Generate()

	if e != nil {
		return e
	}

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

func supportedOrchestrators() map[string]orchestrators.Pipeline {
	return map[string]orchestrators.Pipeline{
		"github-actions": new(orchestrators.GitHubActions),
		"cloud-build":    new(orchestrators.CloudBuild),
		"gitlab-ci":      new(orchestrators.GitLabCI),
	}
}

func listSupportedOrchestrators() string {
	var names []string

	for key := range supportedOrchestrators() {
		names = append(names, key)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}
