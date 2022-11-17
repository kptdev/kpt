// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdtree

import (
	"context"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/util/argutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

func GetTreeRunner(ctx context.Context, name string) *TreeRunner {
	r := &TreeRunner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "tree [DIR]",
		Short:   pkgdocs.TreeShort,
		Long:    pkgdocs.TreeLong,
		Example: pkgdocs.TreeExamples,
		RunE:    r.runE,
		Args:    cobra.MaximumNArgs(1),
	}

	r.Command = c
	return r
}

func NewCommand(ctx context.Context, name string) *cobra.Command {
	return GetTreeRunner(ctx, name).Command
}

// TreeRunner contains the run function
type TreeRunner struct {
	Command *cobra.Command
	Ctx     context.Context
}

func (r *TreeRunner) runE(c *cobra.Command, args []string) error {
	var input kio.Reader
	var root = "."
	if len(args) == 0 {
		args = append(args, root)
	}
	root = filepath.Clean(args[0])
	resolvedPath, err := argutil.ResolveSymlink(r.Ctx, args[0])
	if err != nil {
		return err
	}
	input = kio.LocalPackageReader{
		PackagePath:       resolvedPath,
		MatchFilesGlob:    r.getMatchFilesGlob(),
		PreserveSeqIndent: true,
		WrapBareSeqNode:   true,
	}
	fltrs := []kio.Filter{&filters.IsLocalConfig{
		IncludeLocalConfig: true,
	}}

	return runner.HandleError(r.Ctx, kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: fltrs,
		Outputs: []kio.Writer{TreeWriter{
			Root:   root,
			Writer: printer.FromContextOrDie(r.Ctx).OutStream(),
		}},
	}.Execute())
}

func (r *TreeRunner) getMatchFilesGlob() []string {
	return append([]string{kptfilev1.KptFileName}, kio.DefaultMatch...)
}
