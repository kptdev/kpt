// Copyright 2019 The Kubernetes Authors.
// Copyright 2026 The kpt Authors
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

package cmdsink

import (
	"context"

	"github.com/kptdev/kpt/internal/docs/generated/fndocs"
	"github.com/kptdev/kpt/pkg/lib/util/cmdutil"
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
