// Copyright 2019 The kpt Authors
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

package diff

import (
	"context"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:          "diff [PKG_PATH@VERSION] [flags]",
		Short:        pkgdocs.DiffShort,
		Long:         pkgdocs.DiffShort + "\n" + pkgdocs.DiffLong,
		Example:      pkgdocs.DiffExamples,
		PreRunE:      r.preRunE,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	diffTool := "diff"
	if tool := os.Getenv("KPT_EXTERNAL_DIFF"); tool != "" {
		diffTool = tool
	}
	diffToolOpts := os.Getenv("KPT_EXTERNAL_DIFF_OPTS")
	c.Flags().StringVar(&r.diffType, "diff-type", "",
		"diff type you want to perform e.g. "+diff.SupportedDiffTypesLabel())
	c.Flags().StringVar(&r.DiffTool, "diff-tool", diffTool,
		"diff tool to use to show the changes")
	c.Flags().StringVar(&r.DiffToolOpts, "diff-tool-opts", diffToolOpts,
		"diff tool commandline options to use to show the changes")
	c.Flags().BoolVar(&r.Debug, "debug", false,
		"when true, prints additional debug information and do not delete staged pkg dirs")
	r.C = c
	r.Output = printer.FromContextOrDie(r.ctx).OutStream()
	cmdutil.FixDocs("kpt", parent, c)
	return r
}

// NewCommand returns a diff command instance.
func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).C
}

// Runner contains the run function
type Runner struct {
	ctx context.Context
	diff.Command
	C        *cobra.Command
	diffType string
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = append(args, pkg.CurDir)
	}
	dirVer := args[0]
	dir, version, err := argutil.ParseDirVersion(dirVer)
	if err != nil {
		return err
	}
	if r.diffType == "" {
		// pick sensible defaults for diff-type
		r.DiffType = diff.TypeLocal
		if version != "" {
			// if target version is specified, default to 'combined' diff-type.
			// xref: https://github.com/GoogleContainerTools/kpt/issues/139
			r.DiffType = diff.TypeCombined
		}
	} else {
		r.DiffType = diff.Type(r.diffType)
	}

	resolvedPath, err := argutil.ResolveSymlink(r.ctx, dir)
	if err != nil {
		return err
	}

	absResolvedPath, _, err := pathutil.ResolveAbsAndRelPaths(resolvedPath)
	if err != nil {
		return err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absResolvedPath)
	if err != nil {
		return err
	}
	r.Path = string(p.UniquePath)
	r.Ref = version
	r.Output = printer.FromContextOrDie(r.ctx).OutStream()

	return r.Validate()
}

func (r *Runner) runE(_ *cobra.Command, _ []string) error {
	return r.Run(r.ctx)
}
