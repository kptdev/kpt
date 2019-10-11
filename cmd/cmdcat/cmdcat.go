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

// Package cmdcat contains the cat command
package cmdcat

import (
	"fmt"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/yaml"
)

// Cmd returns a command runner.
func Cmd() *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:   "cat DIR...",
		Short: "Print Resource Config from a local package",
		Long: `Print Resource Config from a local package.

  DIR:
    Path to local package directory.
`,
		Example: `# print Resource config from a package
kpt cat my-package/

# wrap Resource config from a package in an ResourceList
kpt cat my-package/ --wrap-kind ResourceList --wrap-version kpt.dev/v1alpha1 --function-config fn.yaml

# unwrap Resource config from a package in an ResourceList
... | kpt cat

# write as json
kpt cat my-package
`,
		RunE:         r.runE,
		SilenceUsage: true,
	}
	c.Flags().BoolVar(&r.IncludeSubpackages, "include-subpackages", true,
		"also print resources from subpackages.")
	c.Flags().BoolVar(&r.Format, "format", true,
		"format resource config yaml before printing.")
	c.Flags().BoolVar(&r.KeepAnnotations, "annotate", false,
		"annotate resources with their file origins.")
	c.Flags().StringVar(&r.WrapKind, "wrap-kind", "",
		"if set, wrap the output in this list type kind.")
	c.Flags().StringVar(&r.WrapApiVersion, "wrap-version", "",
		"if set, wrap the output in this list type apiVersion.")
	c.Flags().StringVar(&r.FunctionConfig, "function-config", "",
		"path to function config to put in ResourceList -- only if wrapped in a ResourceList.")
	c.Flags().StringSliceVar(&r.Styles, "style", []string{},
		"yaml styles to apply.  may be 'TaggedStyle', 'DoubleQuotedStyle', 'LiteralStyle', "+
			"'FoldedStyle', 'FlowStyle'.")
	c.Flags().BoolVar(&r.StripComments, "strip-comments", false,
		"remove comments from yaml.")
	c.Flags().BoolVar(&r.IncludeReconcilers, "include-reconcilers", false,
		"if true, include reconciler Resources in the output.")
	c.Flags().BoolVar(&r.ExcludeNonReconcilers, "exclude-non-reconcilers", false,
		"if true, exclude non-reconciler Resources in the output.")
	r.CobraCommand = c
	return r
}

// Runner contains the run function
type Runner struct {
	IncludeSubpackages    bool
	Format                bool
	KeepAnnotations       bool
	WrapKind              string
	WrapApiVersion        string
	FunctionConfig        string
	Styles                []string
	StripComments         bool
	IncludeReconcilers    bool
	ExcludeNonReconcilers bool
	CobraCommand          *cobra.Command
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	// if there is a function-config specified, emit it
	var functionConfig *yaml.RNode
	if r.FunctionConfig != "" {
		configs, err := kio.LocalPackageReader{PackagePath: r.FunctionConfig,
			OmitReaderAnnotations: !r.KeepAnnotations}.Read()
		if err != nil {
			return err
		}
		if len(configs) != 1 {
			return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
		}
		functionConfig = configs[0]
	}

	var inputs []kio.Reader
	for _, a := range args {
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath:        a,
			IncludeSubpackages: r.IncludeSubpackages,
		})
	}
	if len(inputs) == 0 {
		inputs = append(inputs, &kio.ByteReader{Reader: c.InOrStdin()})
	}
	var fltr []kio.Filter
	// don't include reconcilers
	fltr = append(fltr, &filters.IsReconcilerFilter{
		ExcludeReconcilers:    !r.IncludeReconcilers,
		IncludeNonReconcilers: !r.ExcludeNonReconcilers,
	})
	if r.Format {
		fltr = append(fltr, filters.FormatFilter{})
	}
	if r.StripComments {
		fltr = append(fltr, filters.StripCommentsFilter{})
	}

	var outputs []kio.Writer
	outputs = append(outputs, kio.ByteWriter{
		Writer:                c.OutOrStdout(),
		KeepReaderAnnotations: r.KeepAnnotations,
		WrappingKind:          r.WrapKind,
		WrappingApiVersion:    r.WrapApiVersion,
		FunctionConfig:        functionConfig,
		Style:                 yaml.GetStyle(r.Styles...),
	})

	return kio.Pipeline{Inputs: inputs, Filters: fltr, Outputs: outputs}.Execute()
}
