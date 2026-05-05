// Copyright 2019,2026 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdcat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kptdev/kpt/internal/docs/generated/pkgdocs"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

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
		"format resource config yaml before printing.")
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

	if info, err := os.Stat(args[0]); err == nil && !info.IsDir() {
		switch strings.ToLower(filepath.Ext(args[0])) {
		case ".yaml", ".yml", ".json":
		default:
			if filepath.Base(args[0]) == kptfilev1.KptFileName {
				return fmt.Errorf("%q is a %s package metadata file, not a resource file; no resources will be read", args[0], kptfilev1.KptFileName)
			}
			return fmt.Errorf("%q is not a YAML/JSON file; no resources will be read", args[0])
		}
	}

	out := &bytes.Buffer{}
	e := runner.ExecuteCmdOnPkgs{
		Writer:             out,
		NeedOpenAPI:        false,
		RecurseSubPackages: r.RecurseSubPackages,
		CmdRunner:          r,
		RootPkgPath:        filepath.Clean(args[0]),
		SkipPkgPathPrint:   true,
	}

	err := e.Execute()
	if err != nil {
		return err
	}

	res := strings.TrimSuffix(out.String(), "---\n")
	res = strings.TrimSuffix(res, "---")
	fmt.Fprintf(writer, "%s", res)

	return nil
}

func (r *CatRunner) ExecuteCmd(w io.Writer, pkgPath string) error {
	input := kio.LocalPackageReader{PackagePath: pkgPath, PackageFileName: kptfilev1.KptFileName}
	out := &bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: r.catFilters(),
		Outputs: r.outputWriter(out),
	}.Execute()

	if err != nil {
		// Wrap with package context so the user knows which package failed;
		// the root command's error handler is responsible for printing.
		return fmt.Errorf("kpt pkg cat: %q: %w", pkgPath, err)
	}
	outStr := out.String()
	fmt.Fprint(w, outStr)
	if outStr != "" {
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
