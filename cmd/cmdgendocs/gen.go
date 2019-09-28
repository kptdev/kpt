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
	"kpt.dev/util/duck"
)

var noop = func(cmd *cobra.Command, args []string) {}

func getHelpTopics(cmd *cobra.Command, dir string) error {
	var tutorials *cobra.Command
	var apis *cobra.Command
	for _, c := range cmd.Commands() {
		c.DisableAutoGenTag = true
		if c.Use == "tutorials" {
			tutorials = c
			// so it gets picked up for gen
			c.Run = noop
			for _, c := range c.Commands() {
				c.Run = noop
			}
			cmd.RemoveCommand(tutorials)
		}
		if c.Use == "apis" {
			apis = c
			// so it gets picked up for gen
			c.Run = noop
			for _, c := range c.Commands() {
				c.Run = noop
			}
			cmd.RemoveCommand(apis)
		}
	}

	if tutorials != nil {
		if err := doc.GenMarkdownTree(tutorials, dir); err != nil {
			return err
		}
	}
	if apis != nil {
		if err := doc.GenMarkdownTree(apis, dir); err != nil {
			return err
		}
	}

	// do duck-typed commands
	cmd = duck.HelpCommand
	cmd.DisableAutoGenTag = true
	if err := duck.AddCommands("", cmd); err != nil {
		return err
	}
	// move the commands up a level
	for _, c := range cmd.Commands() {
		if c.Name() == "get" || c.Name() == "set" {
			cmd.RemoveCommand(c)
			for _, s := range c.Commands() {
				s.Use = c.Name() + "-" + s.Use
			}
			cmd.AddCommand(c.Commands()...)
		}
	}
	if err := doc.GenMarkdownTree(cmd, dir); err != nil {
		return err
	}

	return nil
}
