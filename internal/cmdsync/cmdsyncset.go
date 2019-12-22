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
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/sync"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewSetRunner(parent string) *SetRunner {
	r := &SetRunner{}
	c := &cobra.Command{
		Use:     "set REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY",
		RunE:    r.runE,
		Long:    docs.SyncSetLong,
		Short:   docs.SyncSetShort,
		Example: docs.SyncSetExamples,
		Args:    cobra.ExactArgs(2),
		PreRunE: r.preRunE,
	}

	c.Flags().StringVar(&r.Strategy, "strategy", "", "update strategy to use.")
	c.Flags().BoolVar(&r.Dependency.EnsureNotExists, "prune", false,
		"prune the dependency when it is synced.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewSetCommand(parent string) *cobra.Command {
	return NewSetRunner(parent).Command
}

type SetRunner struct {
	Dependency kptfile.Dependency
	Command    *cobra.Command
	Strategy   string
}

func (r *SetRunner) preRunE(_ *cobra.Command, args []string) error {
	t, err := parse.GitParseArgs(args)
	if err != nil {
		return err
	}
	r.Dependency.Git = t.Git
	r.Dependency.Name = args[1]
	return nil
}

func (r *SetRunner) runE(_ *cobra.Command, args []string) error {
	return sync.SetDependency(r.Dependency)
}
