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

// Package cmdsub contains the sub command.
package cmdsub

import (
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "sub PKG_DIR SUBSTITUTION_NAME NEW_VALUE",
		Args:    cobra.RangeArgs(1, 3),
		Short:   docs.SubShort,
		Long:    docs.SubLong,
		Example: docs.SubExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	c.Flags().BoolVar(&r.Substitute.Override, "override", false,
		"override previously substituted values")
	c.Flags().BoolVar(&r.Substitute.Revert, "revert", false,
		"override previously substituted values")
	cmdutil.FixDocs("kpt", parent, c)

	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

type Runner struct {
	Command    *cobra.Command
	Help       Help
	Substitute Substitute
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) < 3 && !(r.Substitute.Revert && len(args) == 2) {
		return cmdutil.HandleError(c, r.Help.preRunE(c, args))
	}
	return cmdutil.HandleError(c, r.Substitute.preRunE(c, args))
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) < 3 && !(r.Substitute.Revert && len(args) == 2) {
		return cmdutil.HandleError(c, r.Help.runE(c, args))
	}
	return cmdutil.HandleError(c, r.Substitute.runE(c, args))
}
