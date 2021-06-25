// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdcat

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/pkgdocs"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetCatRunner returns a command CatRunner.
func GetCatRunner(name string) *CatRunner {
	r := &CatRunner{}
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

func NewCommand(name string) *cobra.Command {
	return GetCatRunner(name).Command
}

// CatRunner contains the run function
type CatRunner struct {
	Format             bool
	KeepAnnotations    bool
	Styles             []string
	StripComments      bool
	Command            *cobra.Command
	RecurseSubPackages bool
}

func (r *CatRunner) runE(c *cobra.Command, args []string) error {
	var writer = c.OutOrStdout()
	if len(args) == 0 {
		args = append(args, ".")
	}

	out := &bytes.Buffer{}
	e := runner.ExecuteCmdOnPkgs{
		Writer:             out,
		NeedOpenAPI:        false,
		RecurseSubPackages: r.RecurseSubPackages,
		CmdRunner:          r,
		RootPkgPath:        args[0],
		SkipPkgPathPrint:   true,
	}

	err := e.Execute()
	if err != nil {
		return err
	}

	res := strings.TrimSuffix(out.String(), "---")
	fmt.Fprintf(writer, "%s", res)

	return nil
}

func (r *CatRunner) ExecuteCmd(w io.Writer, pkgPath string) error {
	input := kio.LocalPackageReader{PackagePath: pkgPath, PackageFileName: kptfilev1.KptFileName}
	out := &bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{input},
		Filters: r.catFilters(),
		Outputs: r.out(out),
	}.Execute()

	if err != nil {
		// return err if there is only package
		if r.RecurseSubPackages {
			// print error message and continue if there are multiple packages to cat
			fmt.Fprintf(w, "%s in package %q\n", err.Error(), pkgPath)
		}
		return err
	}
	fmt.Fprint(w, out.String())
	if out.String() != "" {
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

func (r *CatRunner) out(w io.Writer) []kio.Writer {
	var outputs []kio.Writer
	var functionConfig *yaml.RNode

	// remove this annotation explicitly, the ByteWriter won't clear it by
	// default because it doesn't set it
	clear := []string{"config.kubernetes.io/path"}
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
