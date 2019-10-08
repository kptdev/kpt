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

// Package cmdxargs contains the cmdxargs command
package cmdwrap

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"kpt.dev/cmdreconcile/cmdxargs"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "wrap CMD...",
		Short: "Wrap a reconcile command in xargs and pipe to merge + fmt",
		Long: `Wrap a reconcile command in xargs and pipe to merge + fmt.

Porcelain for running CMD wrapped in 'kpt xargs' and piping the result to
'kpt merge | kpt fmt --set-filenames'.

If KPT_OVERRIDE_PKG is set to a directory in the container, wrap will also read
the contents of the override package directory and merge them on top of the CMD
output.
`,
		Example: `

`,
		RunE:               r.runE,
		SilenceUsage:       true,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Args:               cobra.MinimumNArgs(1),
	}
	r.C = c
	r.Xargs = cmdxargs.Cmd()
	c.Flags().BoolVar(&r.Xargs.EnvOnly, "env-only", true, "only set env vars, not arguments.")
	c.Flags().StringVar(&r.Xargs.WrapKind, "wrap-kind", "List", "wrap the input xargs give to the command in this type.")
	c.Flags().StringVar(&r.Xargs.WrapVersion, "wrap-version", "v1", "wrap the input xargs give to the command in this type.")
	return r
}

// Runner contains the run function
type Runner struct {
	C      *cobra.Command
	Xargs  *cmdxargs.Runner
	getEnv func(key string) string
}

const (
	KptMerge       = "KPT_MERGE"
	KptOverridePkg = "KPT_OVERRIDE_PKG"
)

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if r.getEnv == nil {
		r.getEnv = os.Getenv
	}
	xargsIn := &bytes.Buffer{}
	if _, err := io.Copy(xargsIn, c.InOrStdin()); err != nil {
		return err
	}
	mergeInput := bytes.NewBuffer(xargsIn.Bytes())
	// Run the reconciler
	xargsOut := &bytes.Buffer{}
	r.Xargs.C.SetArgs(args)
	r.Xargs.C.SetIn(xargsIn)
	r.Xargs.C.SetOut(xargsOut)
	r.Xargs.C.SetErr(os.Stderr)
	if err := r.Xargs.C.Execute(); err != nil {
		return err
	}

	// merge the results
	buff := &kio.PackageBuffer{}

	var fltrs []kio.Filter
	var inputs []kio.Reader
	if r.getEnv(KptMerge) == "" || r.getEnv(KptMerge) == "true" || r.getEnv(KptMerge) == "1" {
		inputs = append(inputs, &kio.ByteReader{Reader: mergeInput})
		fltrs = append(fltrs, &filters.MergeFilter{})
	}
	inputs = append(inputs, &kio.ByteReader{Reader: xargsOut})

	if err := (kio.Pipeline{Inputs: inputs, Filters: fltrs, Outputs: []kio.Writer{buff}}).
		Execute(); err != nil {
		return err
	}

	inputs, fltrs = []kio.Reader{buff}, nil
	if r.getEnv(KptOverridePkg) != "" {
		// merge the overrides on top of the output
		fltrs = append(fltrs, filters.MergeFilter{})
		inputs = append(inputs,
			kio.LocalPackageReader{
				OmitReaderAnnotations: true, // don't set path annotations, as they would override
				PackagePath:           r.getEnv(KptOverridePkg)})
	}
	fltrs = append(fltrs,
		&filters.FileSetter{
			FilenamePattern: filepath.Join("config", filters.DefaultFilenamePattern)},
		&filters.FormatFilter{})

	err := kio.Pipeline{
		Inputs:  inputs,
		Filters: fltrs,
		Outputs: []kio.Writer{kio.ByteWriter{
			Sort:                  true,
			KeepReaderAnnotations: true,
			Writer:                c.OutOrStdout(),
			WrappingKind:          kio.ResourceListKind,
			WrappingApiVersion:    kio.ResourceListApiVersion}}}.Execute()
	if err != nil {
		return err
	}

	return nil
}
