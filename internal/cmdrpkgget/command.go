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
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/get"
	"k8s.io/kubectl/pkg/cmd/util"
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

--revision
  Revision of the package to get. Any package whose revision matches this value will be included in the results.

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
		Example:    "kpt alpha rpkg get repository:package:v1 --namespace=default",
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     porch.HidePorchCommands,
	}
	r.Command = c

	// Create flags
	c.Flags().StringVar(&r.name, "name", "", "Name of the packages to get. Any package whose name contains this value will be included in the results.")
	c.Flags().StringVar(&r.revision, "revision", "", "Revision of the packages to get. Any package whose revision matches this value will be included in the results.")
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
	revision   string
	printFlags *get.PrintFlags

	requestTable bool
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	outputOption := cmd.Flags().Lookup("output").Value.String()
	if strings.Contains(outputOption, "custom-columns") || outputOption == "yaml" || strings.Contains(outputOption, "json") {
		r.requestTable = false
	} else {
		r.requestTable = true
	}

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

	f := util.NewFactory(r.cfg)

	b := f.NewBuilder().
		Unstructured().
		NamespaceParam(*r.cfg.Namespace).DefaultNamespace()

	if len(args) > 0 {
		b = b.ResourceNames("packagerevisions", args...)
	} else {
		b = b.SelectAllParam(true).
			ResourceTypes("packagerevisions")
	}

	b = b.ContinueOnError().
		Latest().
		Flatten()

	if r.requestTable {
		b = b.TransformRequests(func(req *rest.Request) {
			req.SetHeader("Accept", strings.Join([]string{
				"application/json;as=Table;g=meta.k8s.io;v=v1",
				"application/json",
			}, ","))
		})
	}

	res := b.Do()
	if err := res.Err(); err != nil {
		return errors.E(op, err)
	}

	infos, err := res.Infos()
	if err != nil {
		return errors.E(op, err)
	}

	for _, i := range infos {
		if r.match(i.Object) {
			objs = append(objs, i.Object)
		}
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return errors.E(op, err)
	}
	if r.requestTable {
		printer = &get.TablePrinter{
			Delegate: printer,
		}
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

func (r *runner) match(obj runtime.Object) bool {
	unstr, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return true // matches
	}
	packageName, _, _ := unstructured.NestedString(unstr.Object, "spec", "packageName")
	revision, _, _ := unstructured.NestedString(unstr.Object, "spec", "revision")

	if !strings.Contains(packageName, r.name) {
		return false
	}
	if r.revision != "" && r.revision != revision {
		return false
	}
	return true
}
