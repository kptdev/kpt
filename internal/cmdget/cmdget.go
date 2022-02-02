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

// Package cmdget contains the get command
package cmdget

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
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// NewRunner returns a command runner
func NewRunner(ctx context.Context, parent string) *Runner {
	r := &Runner{
		ctx: ctx,
	}
	c := &cobra.Command{
		Use:        "get {REPO_URI[.git]/PKG_PATH[@VERSION]|IMAGE:TAG} [LOCAL_DEST_DIRECTORY]",
		Args:       cobra.MinimumNArgs(1),
		Short:      docs.GetShort,
		Long:       docs.GetShort + "\n" + docs.GetLong,
		Example:    docs.GetExamples,
		RunE:       r.runE,
		PreRunE:    r.preRunE,
		SuggestFor: []string{"clone", "cp", "fetch"},
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	c.Flags().StringVar(&r.strategy, "strategy", string(kptfilev1.ResourceMerge),
		"update strategy that should be used when updating this package -- must be one of: "+
			strings.Join(kptfilev1.UpdateStrategiesAsStrings(), ","))
	_ = c.RegisterFlagCompletionFunc("strategy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return kptfilev1.UpdateStrategiesAsStrings(), cobra.ShellCompDirectiveDefault
	})
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function
type Runner struct {
	ctx      context.Context
	Get      get.Command
	Command  *cobra.Command
	strategy string
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdget.preRunE"
	if len(args) == 1 {
		args = append(args, pkg.CurDir)
	} else {
		_, err := os.Lstat(args[1])
		if err == nil || os.IsExist(err) {
			resolvedPath, err := argutil.ResolveSymlink(r.ctx, args[1])
			if err != nil {
				return errors.E(op, err)
			}
			args[1] = resolvedPath
		}
	}

	upstream, err := location.ParseReference(
		args[0],
		location.WithContext(r.ctx),
		location.WithParsers(location.GitParser, location.OciParser))
	if err != nil {
		return err
	}
	r.Get.Upstream = upstream

	destination, err := getDest(args[1], upstream)
	if err != nil {
		return err
	}

	absDestPath, _, err := pathutil.ResolveAbsAndRelPaths(destination)
	if err != nil {
		return err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absDestPath)
	if err != nil {
		return errors.E(op, types.UniquePath(destination), err)
	}
	r.Get.Destination = string(p.UniquePath)

	strategy, err := kptfilev1.ToUpdateStrategy(r.strategy)
	if err != nil {
		return err
	}
	r.Get.UpdateStrategy = strategy
	return nil
}

func getDest(dir string, ref location.Reference) (string, error) {
	destination := filepath.Clean(dir)

	f, err := os.Stat(destination)
	if os.IsNotExist(err) {
		parent := filepath.Dir(destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			// error -- fetch to directory where parent does not exist
			return "", fmt.Errorf("parent directory %q does not exist", parent)
		}
		// fetch to a specific directory -- don't default the name
		return destination, nil
	}

	if !f.IsDir() {
		return "", fmt.Errorf("LOCAL_PKG_DEST must be a directory")
	}

	if name, ok := location.DefaultDirectoryName(ref); ok {
		return filepath.Join(destination, name), nil
	}

	// this reference type does not provide a default name.
	// the error message is a prompt to provide complete path to new dir.
	return "", fmt.Errorf("destination directory already exists")
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	const op errors.Op = "cmdget.runE"
	if err := r.Get.Run(r.ctx); err != nil {
		return errors.E(op, types.UniquePath(r.Get.Destination), err)
	}

	return nil
}
