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

// Package cmdfix contains the fix command
package cmdfix

import (
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/fix"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/fieldmeta"
)

// NewRunner returns a command runner
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:        "fix LOCAL_PKG_DIR",
		Args:       cobra.ExactArgs(1),
		Short:      docs.FixShort,
		Long:       docs.FixShort + "\n" + docs.FixLong,
		Example:    docs.FixExamples,
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		SuggestFor: []string{"upgrade", "migrate"},
	}
	cmdutil.FixDocs("kpt", parent, c)
	c.Flags().BoolVar(&r.Fix.DryRun, "dry-run", false,
		`Dry run emits the actions`)
	r.Command = c
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function
type Runner struct {
	Command *cobra.Command
	Fix     fix.Command
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	r.Fix.PkgPath = args[0]
	r.Fix.StdOut = c.OutOrStdout()

	// require package is checked into git before trying to fix it
	g := gitutil.NewLocalGitRunner(args[0])
	if err := g.Run("status", "-s"); err != nil {
		return errors.Errorf(
			"kpt packages must be tracked by git before making fix to revert unwanted changes: %v", err)
	}
	if strings.TrimSpace(g.Stdout.String()) != "" {
		return errors.Errorf("must commit package changes to git %s before attempting to fix it",
			args[0])
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	fieldmeta.SetShortHandRef("$kpt-set")
	return r.Fix.Run()
}
