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

package cmdrpkgget

import (
	"context"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/kubectl/pkg/cmd/get"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgget"
	longMsg = `
kpt alpha rpkg get [PACKAGE ...] [flags]

Args:

PACKAGE:
  Name of the package revision to get.

Flags:

--name
  Name of the packages to get. Any package whose name contains this value will be included in the results.

`
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:        ctx,
		cfg:        rcg,
		printFlags: get.NewGetPrintFlags(),
	}
	c := &cobra.Command{
		Use:        "get",
		Aliases:    []string{"list"},
		SuggestFor: []string{},
		Short:      "Gets or lists packages in registered repositories.",
		Long:       longMsg,
		Example:    "TODO",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags
	c.Flags().StringVar(&r.name, "name", "", "Name of the packages to get. Any package whose name contains this value will be included in the results.")
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
	name       string
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
		for _, pkg := range args {
			pr := &porchapi.PackageRevision{}
			if err := r.client.Get(r.ctx, client.ObjectKey{
				Namespace: *r.cfg.Namespace,
				Name:      pkg,
			}, pr); err != nil {
				return errors.E(op, err)
			}

			// TODO: is the server not returning GVK?
			pr.Kind = "PackageRevision"
			pr.APIVersion = porchapi.SchemeGroupVersion.Identifier()

			objs = append(objs, pr)
		}
	} else {
		var list porchapi.PackageRevisionList
		if err := r.client.List(r.ctx, &list); err != nil {
			return errors.E(op, err)
		}
		for i := range list.Items {
			pr := &list.Items[i]
			if r.match(pr) {
				objs = append(objs, pr)
			}
		}
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

func (r *runner) match(pr *porchapi.PackageRevision) bool {
	return strings.Contains(pr.Name, r.name)
}
