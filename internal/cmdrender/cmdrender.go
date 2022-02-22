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
	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	"github.com/GoogleContainerTools/kpt/internal/util/render"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
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
	}
	c.Flags().StringVar(&r.resultsDirPath, "results-dir", "",
		"path to a directory to save function results")
	c.Flags().StringVarP(&r.dest, "output", "o", "",
		fmt.Sprintf("output resources are written to provided location. Allowed values: %s|%s|<OUT_DIR_PATH>", cmdutil.Stdout, cmdutil.Unwrap))
	c.Flags().StringVar(&r.imagePullPolicy, "image-pull-policy", string(fnruntime.IfNotPresentPull),
		fmt.Sprintf("pull image before running the container. It must be one of %s, %s and %s.", fnruntime.AlwaysPull, fnruntime.IfNotPresentPull, fnruntime.NeverPull))
	c.Flags().BoolVar(&r.allowExec, "allow-exec", false,
		"allow binary executable to be run during pipeline execution.")
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
	allowExec       bool
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
	var err error
	r.pkgPath, err = argutil.ResolveSymlink(r.ctx, r.pkgPath)
	if err != nil {
		return err
	}
	if r.dest != "" && r.dest != cmdutil.Stdout && r.dest != cmdutil.Unwrap {
		if err := cmdutil.CheckDirectoryNotPresent(r.dest); err != nil {
			return err
		}
	}
	if r.resultsDirPath != "" {
		err := os.MkdirAll(r.resultsDirPath, 0755)
		if err != nil {
			return fmt.Errorf("cannot read or create results dir %q: %w", r.resultsDirPath, err)
		}
	}
	return cmdutil.ValidateImagePullPolicyValue(r.imagePullPolicy)
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	var output io.Writer
	outContent := bytes.Buffer{}
	if r.dest != "" {
		// this means the output should be written to another destination
		// capture the content to be written
		output = &outContent
	}
	absPkgPath, _, err := pathutil.ResolveAbsAndRelPaths(r.pkgPath)
	if err != nil {
		return err
	}
	executor := render.Renderer{
		PkgPath:         absPkgPath,
		ResultsDirPath:  r.resultsDirPath,
		Output:          output,
		ImagePullPolicy: cmdutil.StringToImagePullPolicy(r.imagePullPolicy),
		AllowExec:       r.allowExec,
		FileSystem:      filesys.FileSystemOrOnDisk{},
	}
	if err := executor.Execute(r.ctx); err != nil {
		return err
	}

	return cmdutil.WriteFnOutput(r.dest, outContent.String(), false, printer.FromContextOrDie(r.ctx).OutStream())
}
