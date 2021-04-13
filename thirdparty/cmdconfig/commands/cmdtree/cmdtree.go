// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdtree

import (
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

func GetTreeRunner(name string) *TreeRunner {
	r := &TreeRunner{}
	c := &cobra.Command{
		Use:     "tree [DIR | -]",
		Short:   pkgdocs.TreeShort,
		Long:    pkgdocs.TreeLong,
		Example: pkgdocs.TreeExamples,
		RunE:    r.runE,
		Args:    cobra.MaximumNArgs(1),
	}

	r.Command = c
	return r
}

func NewCommand(name string) *cobra.Command {
	return GetTreeRunner(name).Command
}

// TreeRunner contains the run function
type TreeRunner struct {
	Command *cobra.Command
}

func (r *TreeRunner) runE(c *cobra.Command, args []string) error {
	var input kio.Reader
	var root = "."
	if len(args) == 0 {
		args = append(args, root)
	}
	if args[0] == "-" {
		input = &kio.ByteReader{Reader: c.InOrStdin()}
	} else {
		root = filepath.Clean(args[0])
		input = kio.LocalPackageReader{PackagePath: args[0], MatchFilesGlob: r.getMatchFilesGlob()}
	}

	fltrs := []kio.Filter{&filters.IsLocalConfig{
		IncludeLocalConfig: true,
	}}

	return runner.HandleError(c, kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: fltrs,
		Outputs: []kio.Writer{TreeWriter{
			Root:   root,
			Writer: c.OutOrStdout(),
		}},
	}.Execute())
}

func (r *TreeRunner) getMatchFilesGlob() []string {
	return append([]string{kptfilev1alpha2.KptFileName}, kio.DefaultMatch...)
}
