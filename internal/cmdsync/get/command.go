// Copyright 2022 Google LLC
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

package get

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/cmd/get"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdsync.get"
	longMsg = `
kpt alpha sync get NAME [flags]

Args:

NAME:
  Name of the sync resource. Required argument.
`
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:        ctx,
		cfg:        rcg,
		printFlags: get.NewGetPrintFlags(),
	}
	c := &cobra.Command{
		Use:     "get NAME",
		Aliases: []string{"register"},
		Short:   "Gets a RootSync resource with which package was deployed.",
		Long:    longMsg,
		Example: "kpt alpha sync get sync-resource --namespace=default",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags
	r.printFlags.AddFlags(c)

	return r
}

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	// Flags
	printFlags *get.PrintFlags
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"
	client, err := porch.CreateDynamicClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	if len(args) == 0 {
		return errors.E(op, "NAME is required positional argument")
	}

	name := args[0]
	key := client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      name,
	}
	rs := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "configsync.gke.io/v1beta1",
			"kind":       "RootSync",
		},
	}
	if err := r.client.Get(r.ctx, key, &rs); err != nil {
		return errors.E(op, fmt.Errorf("cannot get %s: %v", key, err))
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return errors.E(op, err)
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())

	if err := printer.PrintObj(&rs, w); err != nil {
		return errors.E(op, err)
	}
	if err := w.Flush(); err != nil {
		return errors.E(op, err)
	}

	return nil
}
