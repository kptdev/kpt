// Copyright 2021 Google LLC
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

// Package cmdget contains the get command
package cmdrender

import (
	"context"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/errors"

	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{ctx: ctx}
	c := &cobra.Command{
		Use:     "render [DIR]",
		Short:   "render",
		Long:    "render",
		Example: "render",
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	r.Command.Flags().BoolVar(&r.disableOutputTruncate, "disable-output-truncate",
		false, "Disable the truncation for function error output")
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function pipeline run command
type Runner struct {
	pkgPath               string
	Command               *cobra.Command
	ctx                   context.Context
	disableOutputTruncate bool
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	const op errors.Op = "fn.preRunE"

	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			return errors.E(op, err)
		}
		r.pkgPath = wd
	} else {
		// resolve and validate the provided path
		r.pkgPath = args[0]
	}
	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	err := cmdutil.DockerCmdAvailable()
	if err != nil {
		return err
	}
	executor := Executor{
		PkgPath:        r.pkgPath,
		TruncateOutput: !r.disableOutputTruncate,
	}
	if err = executor.Execute(r.ctx); err != nil {
		return err
	}
	return nil
}
