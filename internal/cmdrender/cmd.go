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

// Package cmdrender contains the render command
package cmdrender

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	"github.com/spf13/cobra"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{ctx: ctx}
	c := &cobra.Command{
		Use:     "render [PKG_PATH] [flags]",
		Short:   docs.RenderShort,
		Long:    docs.RenderShort + "\n" + docs.RenderLong,
		Example: docs.RenderExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
		PostRun: r.postRun,
	}
	c.Flags().StringVar(&r.resultsDirPath, "results-dir", "",
		"path to a directory to save function results")
	c.Flags().StringVarP(&r.dest, "output", "o", "",
		fmt.Sprintf("location (and format) to write output resources to. One of: %s|%s|<OUT_DIR_PATH>", cmdutil.StdOut, cmdutil.Unwrap))
	c.Flags().StringVar(&r.imagePullPolicy, "image-pull-policy", "always",
		"pull image before running the container. It should be one of always, ifNotPresent and never.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function pipeline run command
type Runner struct {
	pkgPath         string
	resultsDirPath  string
	imagePullPolicy string
	dest            string
	Command         *cobra.Command
	ctx             context.Context
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// no pkg path specified, default to current working dir
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		r.pkgPath = wd
	} else {
		// resolve and validate the provided path
		r.pkgPath = args[0]
	}
	if r.resultsDirPath != "" {
		if _, err := os.Stat(r.resultsDirPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("results-dir %q must exist", r.resultsDirPath)
			}
			return fmt.Errorf("results-dir %q check failed: %w", r.resultsDirPath, err)
		}
	}
	return cmdutil.ValidateImagePullPolicyValue(r.imagePullPolicy)
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	err := cmdutil.DockerCmdAvailable()
	if err != nil {
		return err
	}
	var output io.Writer
	outContent := bytes.Buffer{}
	if r.dest != "" {
		// this means the output should be written to another destination
		// capture the content to be written
		output = &outContent
	}
	executor := Executor{
		PkgPath:         r.pkgPath,
		ResultsDirPath:  r.resultsDirPath,
		Output:          output,
		ImagePullPolicy: cmdutil.StringToImagePullPolicy(r.imagePullPolicy),
	}
	err = executor.Execute(r.ctx)
	if err != nil {
		return err
	}

	return cmdutil.WriteFnOutput(r.dest, outContent.String(), false, c.OutOrStdout())
}

func (r *Runner) postRun(_ *cobra.Command, _ []string) {
	if r.dest != "" {
		// do not format/modify resources in package if output should be written to other dest
		return
	}
	pkgutil.FormatPackage(r.pkgPath)
}
