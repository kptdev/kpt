// Copyright 2022 The kpt Authors
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

package installrg

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/livedocs"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

// NewRunner returns a command runner
func NewRunner(
	ctx context.Context,
	factory cmdutil.Factory,
	ioStreams genericclioptions.IOStreams,
) *Runner {
	r := &Runner{
		ctx:       ctx,
		ioStreams: ioStreams,
		factory:   factory,
	}
	c := &cobra.Command{
		Use:     "install-resource-group",
		RunE:    r.runE,
		PreRunE: r.preRunE,
		Short:   livedocs.InstallResourceGroupShort,
		Long:    livedocs.InstallResourceGroupShort + "\n" + livedocs.InstallResourceGroupLong,
		Example: livedocs.InstallResourceGroupExamples,
	}
	r.Command = c
	return r
}

func NewCommand(
	ctx context.Context,
	factory cmdutil.Factory,
	ioStreams genericclioptions.IOStreams,
) *cobra.Command {
	return NewRunner(ctx, factory, ioStreams).Command
}

// Runner contains the run function
type Runner struct {
	ctx       context.Context
	Command   *cobra.Command
	ioStreams genericclioptions.IOStreams
	factory   cmdutil.Factory
}

func (r *Runner) preRunE(_ *cobra.Command, _ []string) error {
	return nil
}

func (r *Runner) runE(_ *cobra.Command, args []string) error {
	// Validate the number of arguments.
	if len(args) > 0 {
		return fmt.Errorf("too many arguments; install-resource-group takes no arguments")
	}
	fmt.Fprint(r.ioStreams.Out, "installing inventory ResourceGroup CRD...")

	err := (&live.ResourceGroupInstaller{
		Factory: r.factory,
	}).InstallRG(r.ctx)

	if err == nil {
		fmt.Fprintln(r.ioStreams.Out, "success")
	} else {
		fmt.Fprintln(r.ioStreams.Out, "failed")
	}
	return err
}
