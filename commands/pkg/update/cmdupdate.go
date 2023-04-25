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

package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// NewRunner returns a command runner.
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:        "update [PKG_PATH@VERSION] [flags]",
		Short:      docs.UpdateShort,
		Long:       docs.UpdateShort + "\n" + docs.UpdateLong,
		Example:    docs.UpdateExamples,
		RunE:       r.runE,
		Args:       cobra.MaximumNArgs(1),
		PreRunE:    r.preRunE,
		SuggestFor: []string{"rebase", "replace"},
	}

	c.Flags().StringVar(&r.strategy, "strategy", string(kptfilev1.ResourceMerge),
		"the update strategy that will be used when updating the package. This will change "+
			"the default strategy for the package -- must be one of: "+
			strings.Join(kptfilev1.UpdateStrategiesAsStrings(), ","))
	_ = c.RegisterFlagCompletionFunc("strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kptfilev1.UpdateStrategiesAsStrings(), cobra.ShellCompDirectiveDefault
	})
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function.
// TODO, support listing versions
type Runner struct {
	ctx      context.Context
	strategy string
	Update   update.Command
	Command  *cobra.Command
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdupdate.preRunE"
	if len(args) == 0 {
		args = append(args, pkg.CurDir)
	}
	if r.strategy == "" {
		r.Update.Strategy = kptfilev1.ResourceMerge
	} else {
		r.Update.Strategy = kptfilev1.UpdateStrategyType(r.strategy)
	}

	parts := strings.Split(args[0], "@")
	if len(parts) > 2 {
		return errors.E(op, errors.InvalidParam, fmt.Errorf("at most 1 version permitted"))
	}

	resolvedPath, err := argutil.ResolveSymlink(r.ctx, parts[0])
	if err != nil {
		return err
	}
	absResolvedPath, _, err := pathutil.ResolveAbsAndRelPaths(resolvedPath)
	if err != nil {
		return err
	}
	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absResolvedPath)
	if err != nil {
		return errors.E(op, err)
	}

	r.Update.Pkg = p

	// TODO: Make sure we handle this in a centralized library and do
	// this consistently across all commands.
	relPath, err := resolveRelPath(p.UniquePath)
	if err != nil {
		return errors.E(op, p.UniquePath, err)
	}
	if strings.HasPrefix(relPath, pkg.ParentDir) {
		return errors.E(op, p.UniquePath, fmt.Errorf("package path must be under current working directory"))
	}

	if len(parts) > 1 {
		r.Update.Ref = parts[1]
	}
	return nil
}

func (r *Runner) runE(_ *cobra.Command, _ []string) error {
	const op errors.Op = "cmdupdate.runE"
	if err := r.Update.Run(r.ctx); err != nil {
		return errors.E(op, r.Update.Pkg.UniquePath, err)
	}

	return nil
}

func resolveRelPath(path types.UniquePath) (string, error) {
	const op errors.Op = "cmdupdate.resolveRelPath"
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.E(op, errors.IO,
			fmt.Errorf("error looking up current working directory: %w", err))
	}

	relPath, err := filepath.Rel(cwd, path.String())
	if err != nil {
		return "", errors.E(op, errors.IO,
			fmt.Errorf("error resolving the relative path: %w", err))
	}
	return relPath, nil
}
