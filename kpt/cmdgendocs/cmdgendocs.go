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

package cmdgendocs

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func Cmd(cmd *cobra.Command) *cobra.Command {
	cmd.DisableAutoGenTag = true
	return &cobra.Command{
		Use: "gen-docs",
		RunE: func(_ *cobra.Command, args []string) error {
			if err := doc.GenMarkdownTree(cmd, args[0]); err != nil {
				return err
			}
			if err := getHelpTopics(cmd, args[0]); err != nil {
				return err
			}

			return nil
		},
		Args:   cobra.ExactArgs(1),
		Hidden: true,
	}
}
