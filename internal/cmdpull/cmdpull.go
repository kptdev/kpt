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

package cmdpull

import (
	"context"
	"os"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/parse/parseref"
	"github.com/GoogleContainerTools/kpt/internal/util/pathutil"
	"github.com/GoogleContainerTools/kpt/internal/util/pull"
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
		Use:     "pull {REPO_URI[.git]/PKG_PATH[@VERSION]|IMAGE:TAG} [LOCAL_DEST_DIRECTORY]",
		Args:    cobra.MinimumNArgs(1),
		Short:   docs.PullShort,
		Long:    docs.PullShort + "\n" + docs.PullLong,
		Example: docs.PullExamples,
		RunE:    r.runE,
		PreRunE: r.preRunE,
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, parent string) *cobra.Command {
	return NewRunner(ctx, parent).Command
}

// Runner contains the run function
type Runner struct {
	ctx     context.Context
	Pull    pull.Command
	Command *cobra.Command
}

func (r *Runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = "cmdpull.preRunE"
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

	origin, destination, err := parseref.ParseArgs(
		args,
		location.WithContext(r.ctx),
		location.WithParsers(location.GitParser, location.OciParser))
	if err != nil {
		return errors.E(op, err)
	}

	r.Pull.Origin = origin

	absDestPath, _, err := pathutil.ResolveAbsAndRelPaths(destination)
	if err != nil {
		return err
	}

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, absDestPath)
	if err != nil {
		return errors.E(op, types.UniquePath(destination), err)
	}
	r.Pull.Destination = string(p.UniquePath)

	return nil
}

func (r *Runner) runE(c *cobra.Command, _ []string) error {
	const op errors.Op = "cmdpull.runE"
	if err := r.Pull.Run(r.ctx); err != nil {
		return errors.E(op, types.UniquePath(r.Pull.Destination), err)
	}

	return nil
}
