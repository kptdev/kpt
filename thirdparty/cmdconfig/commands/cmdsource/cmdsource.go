// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"fmt"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetSourceRunner returns a command for Source.
func GetSourceRunner(name string) *SourceRunner {
	r := &SourceRunner{
		WrapKind:       kio.ResourceListKind,
		WrapAPIVersion: kio.ResourceListAPIVersion,
	}
	c := &cobra.Command{
		Use:     "source [DIR] [flags]",
		Short:   fndocs.SourceShort,
		Long:    fndocs.SourceShort + "\n" + fndocs.SourceLong,
		Example: fndocs.SourceExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.runE,
	}
	c.Flags().StringVar(&r.FunctionConfig, "fn-config", "",
		"path to function config file.")
	c.Flags().BoolVar(&r.IncludeMetaResources,
		"include-meta-resources", false, "include package meta resources in the command output")
	r.Command = c
	_ = c.MarkFlagFilename("fn-config", "yaml", "json", "yml")
	return r
}

func NewCommand(name string) *cobra.Command {
	return GetSourceRunner(name).Command
}

// SourceRunner contains the run function
type SourceRunner struct {
	WrapKind             string
	WrapAPIVersion       string
	FunctionConfig       string
	Command              *cobra.Command
	IncludeMetaResources bool
}

func (r *SourceRunner) runE(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		// default to current working directory
		args = append(args, ".")
	}
	// if there is a function-config specified, emit it
	var functionConfig *yaml.RNode
	if r.FunctionConfig != "" {
		configs, err := kio.LocalPackageReader{PackagePath: r.FunctionConfig}.Read()
		if err != nil {
			return err
		}
		if len(configs) != 1 {
			return fmt.Errorf("expected exactly 1 functionConfig, found %d", len(configs))
		}
		functionConfig = configs[0]
	}

	var outputs []kio.Writer
	outputs = append(outputs, kio.ByteWriter{
		Writer:                c.OutOrStdout(),
		KeepReaderAnnotations: true,
		WrappingKind:          r.WrapKind,
		WrappingAPIVersion:    r.WrapAPIVersion,
		FunctionConfig:        functionConfig,
	})

	var inputs []kio.Reader
	matchFilesGlob := kio.MatchAll
	if r.IncludeMetaResources {
		matchFilesGlob = append(matchFilesGlob, kptfile.KptFileName)
	}
	for _, a := range args {
		pkgPath, err := filepath.Abs(a)
		if err != nil {
			return fmt.Errorf("cannot convert input path %q to absolute path: %w", a, err)
		}
		functionConfigFilter, err := pkg.FunctionConfigFilterFunc(types.UniquePath(pkgPath), r.IncludeMetaResources)
		if err != nil {
			return err
		}
		inputs = append(inputs, kio.LocalPackageReader{
			PackagePath:    a,
			MatchFilesGlob: matchFilesGlob,
			FileSkipFunc:   functionConfigFilter,
		})
	}

	err := kio.Pipeline{Inputs: inputs, Outputs: outputs}.Execute()
	return runner.HandleError(c, err)
}
