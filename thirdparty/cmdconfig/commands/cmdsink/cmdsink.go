// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package cmdsink

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/fndocs"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/spf13/cobra"
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
		Args:    cobra.MinimumNArgs(1),
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
	if err := cmdutil.CheckDirectoryNotPresent(args[0]); err != nil {
		return err
	}
	return cmdutil.WriteToOutput(c.InOrStdin(), nil, args[0])
}
