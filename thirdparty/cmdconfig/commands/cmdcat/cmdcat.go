// Copyright 2019,2026 The kpt Authors.
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

package cmdcat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kptdev/kpt/internal/docs/generated/pkgdocs"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	argsutil "github.com/kptdev/kpt/pkg/lib/util/args"
	"github.com/kptdev/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// krmMatch is the set of file globs that should be processed by this command,
// including the default KRM resource matches and Kptfile.
var krmMatch = append(append([]string{}, kio.DefaultMatch...), kptfilev1.KptFileName)

// GetCatRunner returns a command CatRunner.
func GetCatRunner(ctx context.Context, _ string) *CatRunner {
	r := &CatRunner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "cat [FILE | DIR]",
		Short:   pkgdocs.CatShort,
		Long:    pkgdocs.CatLong,
		Example: pkgdocs.CatExamples,
		RunE:    r.runE,
		Args:    cobra.MaximumNArgs(1),
	}
	c.Flags().BoolVar(&r.Format, "format", true,
		"format resource config YAML before printing (reorders fields to canonical order).")
	c.Flags().BoolVar(&r.KeepAnnotations, "annotate", false,
		"annotate resources with their file origins.")
	c.Flags().StringSliceVar(&r.Styles, "style", []string{},
		"yaml styles to apply.  may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', "+
			"'FoldedStyle', 'FlowStyle'.")
	c.Flags().BoolVar(&r.StripComments, "strip-comments", false,
		"remove comments from yaml.")
	c.Flags().BoolVarP(&r.RecurseSubPackages, "recurse-subpackages", "R", true,
		"print resources recursively in all the nested subpackages")
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, name string) *cobra.Command {
	return GetCatRunner(ctx, name).Command
}

// CatRunner contains the run function
type CatRunner struct {
	Command            *cobra.Command
	Ctx                context.Context
	Format             bool
	KeepAnnotations    bool
	Styles             []string
	StripComments      bool
	RecurseSubPackages bool
}

func (r *CatRunner) runE(c *cobra.Command, args []string) error {
	var writer = c.OutOrStdout()
	if len(args) == 0 {
		args = append(args, ".")
	}

	if err := r.Ctx.Err(); err != nil {
		return runner.HandleError(r.Ctx, err)
	}

	resolvedPath, err := argsutil.ResolveSymlink(r.Ctx, args[0])
	if err != nil {
		return runner.HandleError(r.Ctx, err)
	}

	out := &bytes.Buffer{}

	// Single file: process directly without package traversal.
	if info, err := os.Stat(resolvedPath); err == nil && !info.IsDir() {
		if err := r.ExecuteCmd(out, resolvedPath); err != nil {
			return runner.HandleError(r.Ctx, err)
		}
	} else {
		e := runner.ExecuteCmdOnPkgs{
			Writer:             out,
			NeedOpenAPI:        false,
			RecurseSubPackages: r.RecurseSubPackages,
			CmdRunner:          r,
			RootPkgPath:        filepath.Clean(resolvedPath),
			SkipPkgPathPrint:   true,
		}
		if err := e.Execute(); err != nil {
			return runner.HandleError(r.Ctx, err)
		}
	}

	res := strings.TrimSuffix(out.String(), "---\n")
	res = strings.TrimSuffix(res, "---")
	fmt.Fprintf(writer, "%s", res)
	return nil
}

// ExecuteCmd outputs the contents of a single package at pkgPath.
// It intentionally does NOT recurse into nested subpackages (directories
// containing a Kptfile). Subpackage recursion is handled by the caller
// (runner.ExecuteCmdOnPkgs) which invokes ExecuteCmd once per package
// when RecurseSubPackages is set. Callers invoking ExecuteCmd directly
// will only see the top-level package content; use ExecuteCmdOnPkgs for
// recursive traversal.
func (r *CatRunner) ExecuteCmd(w io.Writer, pkgPath string) error {
	var parts []string

	err := filepath.WalkDir(pkgPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if r.Ctx.Err() != nil {
			return r.Ctx.Err()
		}

		// Skip symlinks to avoid reading outside the package.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if path != pkgPath {
				// Skip nested subpackages; the caller is responsible for
				// recursing into them via RecurseSubPackages if desired.
				if _, statErr := os.Stat(filepath.Join(path, kptfilev1.KptFileName)); statErr == nil {
					return filepath.SkipDir
				}
			}
			return nil
		}

		relPath, _ := filepath.Rel(pkgPath, path)
		if relPath == "." {
			relPath = filepath.Base(path)
		}
		ext := strings.ToLower(filepath.Ext(path))

		if ext == ".yaml" || ext == ".yml" || ext == ".json" || filepath.Base(path) == kptfilev1.KptFileName {
			// KRM file: read through pipeline with formatting.
			buf := &bytes.Buffer{}
			input := kio.LocalPackageReader{PackagePath: path, PackageFileName: "", MatchFilesGlob: krmMatch}
			pErr := kio.Pipeline{
				Inputs:  []kio.Reader{input},
				Filters: r.catFilters(),
				Outputs: r.outputWriter(buf),
			}.Execute()
			if pErr != nil {
				return fmt.Errorf("kpt pkg cat: %q: %w", relPath, pErr)
			}
			if s := buf.String(); s != "" {
				parts = append(parts, s)
			}
		} else {
			// Non-KRM file: display raw if it's valid UTF-8 text.
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if !utf8.Valid(data) {
				return nil // skip binary files
			}
			content := string(data)
			if !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			parts = append(parts, content)
		}
		return nil
	})
	if err != nil {
		return err
	}

	combined := strings.Join(parts, "---\n")
	fmt.Fprint(w, combined)
	if combined != "" {
		fmt.Fprint(w, "---")
	}
	return nil
}

func (r *CatRunner) catFilters() []kio.Filter {
	var fltrs []kio.Filter
	if r.Format {
		fltrs = append(fltrs, filters.FormatFilter{})
	}
	if r.StripComments {
		fltrs = append(fltrs, filters.StripCommentsFilter{})
	}
	return fltrs
}

func (r *CatRunner) outputWriter(w io.Writer) []kio.Writer {
	var outputs []kio.Writer
	var functionConfig *yaml.RNode

	// remove these annotations explicitly; the ByteWriter won't clear them by
	// default because they were set by the LocalPackageReader, not by it.
	clear := []string{
		"config.kubernetes.io/path",
		"internal.config.kubernetes.io/path",
	}
	if r.KeepAnnotations {
		clear = nil
	}

	outputs = append(outputs, kio.ByteWriter{
		Writer:                w,
		KeepReaderAnnotations: r.KeepAnnotations,
		FunctionConfig:        functionConfig,
		Style:                 yaml.GetStyle(r.Styles...),
		ClearAnnotations:      clear,
	})

	return outputs
}
