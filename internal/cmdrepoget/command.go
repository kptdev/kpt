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

package cmdrepoget

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/cmd/get"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrepoget"
	longMsg = `
kpt alpha repo get [flags]

Lists repositories registered with Package Orchestrator.
`
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:        ctx,
		cfg:        rcg,
		printFlags: get.NewGetPrintFlags(),
	}
	c := &cobra.Command{
		Use:     "get [REPOSITORY]",
		Aliases: []string{"ls", "list"},
		Short:   "Lists repositories registered with Package Orchestrator.",
		Long:    longMsg,
		Example: "kpt alpha repo list --namespace default",
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags
	r.printFlags.AddFlags(c)
	return r
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
	client, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = client
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	var objs []runtime.Object

	if len(args) > 0 {
		for _, repo := range args {
			var repository configapi.Repository
			if err := r.client.Get(r.ctx, client.ObjectKey{
				Namespace: *r.cfg.Namespace,
				Name:      repo,
			}, &repository); err != nil {
				return errors.E(op, err)
			}

			repository.Kind = "Repository"
			repository.APIVersion = configapi.GroupVersion.Identifier()

			objs = append(objs, &repository)
		}
	} else {
		var repositories configapi.RepositoryList
		if err := r.client.List(r.ctx, &repositories, client.InNamespace(*r.cfg.Namespace)); err != nil {
			return errors.E(op, err)
		}
		repositories.Kind = "RepositoryList"
		repositories.APIVersion = configapi.GroupVersion.Identifier()
		objs = append(objs, &repositories)
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return errors.E(op, err)
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())

	for _, obj := range objs {
		if err := printer.PrintObj(obj, w); err != nil {
			return errors.E(op, err)
		}
	}

	if err := w.Flush(); err != nil {
		return errors.E(op, err)
	}

	return nil
}
