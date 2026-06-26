// Copyright 2021,2026 The kpt Authors
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
package render

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	docs "github.com/kptdev/kpt/internal/docs/generated/fndocs"
	"github.com/kptdev/kpt/pkg/lib/kptops"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	argsutil "github.com/kptdev/kpt/pkg/lib/util/args"
	"github.com/kptdev/kpt/pkg/lib/util/cmdutil"
	pathutil "github.com/kptdev/kpt/pkg/lib/util/path"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

const PrefixEnvVar = "KPT_IMAGE_PREFIX" // TODO: check if this is in the right place and if there is any convention I should be following


// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{ctx: ctx}

	// find the image prefix
	// priority order: CLI arg - env var - default
	defaultPrefix := os.Getenv(PrefixEnvVar)
	if defaultPrefix == "" {
		defaultPrefix = runneroptions.GHCRImagePrefix
	}

	c := &cobra.Command{
		Use:     "render [PKG_PATH] [flags]",
		Short:   docs.RenderShort,
		Long:    docs.RenderShort + "\n" + docs.RenderLong,
		Example: docs.RenderExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}

	prefix := *c.Flags().String("image-prefix", defaultPrefix, "The prefix to be used when converting from short path to the full url")
	r.InitDefaults(prefix)
	c.Flags().StringVar(&r.resultsDirPath, "results-dir", "",
		"path to a directory to save function results")
	c.Flags().StringVarP(&r.dest, "output", "o", "",
		fmt.Sprintf("output resources are written to provided location. Allowed values: %s|%s|<OUT_DIR_PATH>", cmdutil.Stdout, cmdutil.Unwrap))

	c.Flags().Var(&r.RunnerOptions.ImagePullPolicy, "image-pull-policy",
		"pull image before running the container "+r.RunnerOptions.ImagePullPolicy.HelpAllowedValues())
	_ = c.RegisterFlagCompletionFunc("image-pull-policy", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return r.RunnerOptions.ImagePullPolicy.AllStrings(), cobra.ShellCompDirectiveDefault
	})

	c.Flags().BoolVar(&r.RunnerOptions.AllowExec, "allow-exec", r.RunnerOptions.AllowExec,
		"allow binary executable to be run during pipeline execution.")
	c.Flags().BoolVar(&r.RunnerOptions.AllowNetwork, "allow-network", false,
		"allow functions to access network during pipeline execution.")
	c.Flags().BoolVar(&r.RunnerOptions.AllowWasm, "allow-alpha-wasm", r.RunnerOptions.AllowWasm,
		"allow wasm to be used during pipeline execution.")
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function pipeline run command
type Runner struct {
	pkgPath        string
	resultsDirPath string
	dest           string
	Command        *cobra.Command
	ctx            context.Context

	RunnerOptions runneroptions.RunnerOptions
}

func (r *Runner) InitDefaults(prefix string) {
	r.RunnerOptions.InitDefaults(prefix)
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
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
	r.pkgPath, err = argsutil.ResolveSymlink(r.ctx, r.pkgPath)
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
	return nil
}

func (r *Runner) runE(_ *cobra.Command, _ []string) error {
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
	executor := kptops.Renderer{
		PkgPath:        absPkgPath,
		ResultsDirPath: r.resultsDirPath,
		Output:         output,
		RunnerOptions:  r.RunnerOptions,
		FileSystem:     filesys.FileSystemOrOnDisk{},
	}
	if _, err := executor.Execute(r.ctx); err != nil {
		return err
	}

	return cmdutil.WriteFnOutput(r.dest, outContent.String(), false, printer.FromContextOrDie(r.ctx).OutStream())
}
