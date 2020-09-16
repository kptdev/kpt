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

package cmdsetdependency

import (
	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/parse"
	"github.com/GoogleContainerTools/kpt/internal/util/sync"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewSetDependencyRunner(parent string) *SetDependencyRunner {
	r := &SetDependencyRunner{}
	c := &cobra.Command{
		Use:     "set-dependency REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY",
		RunE:    r.runE,
		Long:    docs.SetdependencyLong,
		Short:   docs.SetdependencyShort,
		Example: docs.SetdependencyExamples,
		Args:    cobra.ExactArgs(2),
	}

	c.Flags().StringVar(&r.MergeStrategy, "strategy", "", "update strategy to use, default: resource-merge")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewSetDependencyRunner(parent).Command
}

type SetDependencyRunner struct {
	Command       *cobra.Command
	MergeStrategy string
}

func (r *SetDependencyRunner) runE(_ *cobra.Command, args []string) error {
	dependency := kptfile.Dependency{
		Name:     args[1],
		Strategy: r.MergeStrategy,
	}
	t, err := parse.GitParseArgs(args)
	if err != nil {
		return err
	}
	dependency.Git = t.Git
	return sync.SetDependency(dependency)
}
