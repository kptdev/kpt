// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsource

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetSourceRunner returns a command for Source.
func GetSourceRunner(name string) *SourceRunner {
	r := &SourceRunner{}
	c := &cobra.Command{
		Use:     "source [DIR]",
		Short:   fndocs.SourceShort,
		Long:    fndocs.SourceLong,
		Example: fndocs.SourceExamples,
		Args:    cobra.MaximumNArgs(1),
		RunE:    r.runE,
	}
	c.Flags().StringVar(&r.WrapKind, "wrap-kind", kio.ResourceListKind,
		"output using this format.")
	c.Flags().StringVar(&r.WrapAPIVersion, "wrap-version", kio.ResourceListAPIVersion,
		"output using this format.")
	c.Flags().StringVar(&r.FunctionConfig, "function-config", "",
		"path to function config.")
	r.Command = c
	_ = c.MarkFlagFilename("function-config", "yaml", "json", "yml")
	return r
}

func NewCommand(name string) *cobra.Command {
	return GetSourceRunner(name).Command
}

// SourceRunner contains the run function
type SourceRunner struct {
	WrapKind       string
	WrapAPIVersion string
	FunctionConfig string
	Command        *cobra.Command
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
	for _, a := range args {
		inputs = append(inputs, kio.LocalPackageReader{PackagePath: a, MatchFilesGlob: kio.MatchAll})
	}

	err := kio.Pipeline{Inputs: inputs, Outputs: outputs}.Execute()
	return runner.HandleError(c, err)
}
