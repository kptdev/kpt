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

// Package cmddesc contains the desc command
package cmddesc

import (
	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/desc"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner.
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:     "desc [PKG_PATH] [flags]",
		Args:    cobra.MaximumNArgs(1),
		Short:   pkgdocs.DescShort,
		Long:    pkgdocs.DescShort + "\n" + pkgdocs.DescLong,
		Example: pkgdocs.DescExamples,
		RunE:    r.runE,
	}
	r.Command = c
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function
type Runner struct {
	Description desc.Command
	Command     *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = append(args, pkg.CurDir)
	}
	p, err := pkg.New(args[0])
	if err != nil {
		return err
	}
	r.Description.PkgPaths = append(r.Description.PkgPaths, string(p.DisplayPath))
	r.Description.StdOut = c.OutOrStdout()
	return r.Description.Run()
}
