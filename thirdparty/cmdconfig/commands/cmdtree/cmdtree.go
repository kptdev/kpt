// Copyright 2019 The Kubernetes Authors.
// Copyright 2026 The kpt Authors.
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

package cmdtree

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/internal/docs/generated/pkgdocs"
	argsutil "github.com/kptdev/kpt/pkg/lib/util/args"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/kptdev/kpt/thirdparty/cmdconfig/commands/runner"
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
	if err := r.Ctx.Err(); err != nil {
		return err
	}
	var input kio.Reader
	var root = "."
	if len(args) == 0 {
		args = append(args, root)
	}
	root = filepath.Clean(args[0])
	resolvedPath, err := argsutil.ResolveSymlink(r.Ctx, args[0])
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
			Root:        root,
			Writer:      printer.FromContextOrDie(r.Ctx).OutStream(),
			NonKRMFiles: discoverNonKRMFiles(r.Ctx, resolvedPath),
		}},
	}.Execute())
}

// discoverNonKRMFiles walks the package tree and returns filenames
// indexed by their containing directory path relative to root.
// Symlinks are skipped. Files that are successfully rendered as KRM resources
// will be deduplicated by the TreeWriter.
func discoverNonKRMFiles(ctx context.Context, root string) map[string][]string {
	result := map[string][]string{}
	pr := printer.FromContextOrDie(ctx)

	if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(pr.ErrStream(), "[WARN] %s: %v\n", path, err)
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") {
			return nil
		}
		if name == kptfilev1.KptFileName {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			fmt.Fprintf(pr.ErrStream(), "[WARN] %s: %v\n", path, err)
			return nil
		}
		result[rel] = append(result[rel], name)
		return nil
	}); err != nil && ctx.Err() == nil {
		fmt.Fprintf(pr.ErrStream(), "[WARN] failed to walk %s: %v\n", root, err)
	}
	return result
}

func (r *TreeRunner) getMatchFilesGlob() []string {
	return append([]string{kptfilev1.KptFileName}, kio.DefaultMatch...)
}
