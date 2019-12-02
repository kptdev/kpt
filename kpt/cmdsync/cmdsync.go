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

// Package cmdsync contains the sync command
package cmdsync

import (
	"github.com/spf13/cobra"
	"kpt.dev/kpt/generated"
	"kpt.dev/kpt/util/cmdutil"
	"kpt.dev/kpt/util/sync"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "sync LOCAL_PKG_DIR",
		Short:   generated.SyncShort,
		Long:    generated.SyncLong,
		Example: generated.SyncExamples,
		RunE:    r.runE,
		Args:    cobra.ExactArgs(1),
		PreRunE: r.preRunE,
	}

	c.Flags().BoolVar(&r.Sync.Verbose, "verbose", false,
		"print verbose logging information.")
	c.Flags().BoolVar(&r.Sync.DryRun, "dry-run", false,
		"print sync actions without performing them.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

type Runner struct {
	Sync    sync.Command
	Command *cobra.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.Sync.Dir = args[0]
	r.Sync.StdOut = c.OutOrStdout()
	r.Sync.StdErr = c.ErrOrStderr()
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	return r.Sync.Run()
}
