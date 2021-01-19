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

package commands

import (
	"github.com/GoogleContainerTools/kpt/internal/pipeline"
	"github.com/spf13/cobra"
)

// GetPipelineCommand returns a command that implements "kpt pipeline <run|...>" subcommands.
func GetPipelineCommand(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use: "pipeline",
		// TODO(droot): wire docs with docs machinery
		Short:   "Pipeline (coming soon)",
		Long:    "Pipeline (coming soon)",
		Example: "pipeline examples (coming soon)",
		Aliases: []string{"p"}, // TODO(droot): kpt p run ?
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := cmd.Flags().GetBool("help")
			if err != nil {
				return err
			}
			if h {
				return cmd.Help()
			}
			return cmd.Usage()
		},
	}

	cmd.AddCommand(pipeline.NewCommand(name))
	return cmd
}
