// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsink

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/thirdparty/cmdconfig/commands/runner"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetSinkRunner returns a command for Sink.
func GetSinkRunner(ctx context.Context, name string) *SinkRunner {
	r := &SinkRunner{
		Ctx: ctx,
	}
	c := &cobra.Command{
		Use:     "sink DIR [flags]",
		Short:   fndocs.SinkShort,
		Long:    fndocs.SinkShort + "\n" + fndocs.SinkLong,
		Example: fndocs.SinkExamples,
		RunE:    r.runE,
	}
	r.Command = c
	return r
}

func NewCommand(ctx context.Context, name string) *cobra.Command {
	return GetSinkRunner(ctx, name).Command
}

// SinkRunner contains the run function
type SinkRunner struct {
	Command *cobra.Command
	Ctx     context.Context
}

func (r *SinkRunner) runE(c *cobra.Command, args []string) error {
	dir := pkg.CurDir
	if len(args) > 0 {
		dir = args[0]
	}
	outputs := []kio.Writer{&kio.LocalPackageWriter{PackagePath: dir}}

	f := kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		if err := kptfile.AreKRM(nodes); err != nil {
			return nodes, err
		}
		return nodes, nil
	})

	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: c.InOrStdin()}},
		Outputs: outputs,
		Filters: []kio.Filter{f},
	}.Execute()

	return runner.HandleError(r.Ctx, err)
}
