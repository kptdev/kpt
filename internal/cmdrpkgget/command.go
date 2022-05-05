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
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/get"
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
		Example:    "kpt alpha rpkg get package-name --namespace=default",
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
	Command *cobra.Command

	// Flags
	name       string
	revision   string
	printFlags *get.PrintFlags

	requestTable bool
}

func (r *runner) preRunE(cmd *cobra.Command, args []string) error {
	outputOption := cmd.Flags().Lookup("output").Value.String()
	if strings.Contains(outputOption, "custom-columns") || outputOption == "yaml" || strings.Contains(outputOption, "json") {
		r.requestTable = false
	} else {
		r.requestTable = true
	}
	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"

	var objs []runtime.Object
	b := resource.NewBuilder(r.cfg).
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
		switch obj := i.Object.(type) {
		case *unstructured.Unstructured:
			o, err := r.tranformAndMatch(obj)
			if err != nil {
				return errors.E(op, err)
			}
			if o != nil {
				objs = append(objs, o)
			}
		default:
			return errors.E(op, fmt.Sprintf("Unrecognized response %T", obj))
		}
	}

	printer, err := r.printFlags.ToPrinter()
	if err != nil {
		return errors.E(op, err)
	}

	if r.requestTable {
		printer = &get.TablePrinter{Delegate: printer}
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

var tableGVK = metav1.SchemeGroupVersion.WithKind("Table")
var packageRevisionGVK = api.SchemeGroupVersion.WithKind("PackageRevision")

func (r *runner) tranformAndMatch(obj *unstructured.Unstructured) (runtime.Object, error) {
	switch gvk := obj.GroupVersionKind(); gvk {
	case tableGVK:
		return r.matchTable(obj)
	case packageRevisionGVK:
		return r.matchPackageRevision(obj)
	default:
		fmt.Fprintf(r.Command.OutOrStderr(), "Unrecognized response type %s", gvk)
		return obj, nil // unrecognized, attempt to print
	}
}

func (r *runner) matchPackageRevision(o *unstructured.Unstructured) (runtime.Object, error) {
	packageName, _, err := unstructured.NestedString(o.Object, "spec", "packageName")
	if err != nil {
		return nil, err
	}
	revision, _, err := unstructured.NestedString(o.Object, "spec", "revision")
	if err != nil {
		return nil, err
	}
	if !strings.Contains(packageName, r.name) {
		return nil, nil
	}
	if r.revision != "" && r.revision != revision {
		return nil, nil
	}
	return o, nil
}

func findColumn(cols []metav1.TableColumnDefinition, name string) int {
	for i := range cols {
		if cols[i].Name == name {
			return i
		}
	}
	return -1
}

func getStringCell(cells []interface{}, col int) (string, bool) {
	if col < 0 {
		return "", false
	}
	s, ok := cells[col].(string)
	return s, ok
}

func (r *runner) matchTable(u *unstructured.Unstructured) (runtime.Object, error) {
	table := &metav1.Table{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, table); err != nil {
		return nil, err
	}

	filtered := make([]metav1.TableRow, 0, len(table.Rows))
	nameCol := findColumn(table.ColumnDefinitions, "Package")
	revisionCol := findColumn(table.ColumnDefinitions, "Revision")

	for i := range table.Rows {
		row := &table.Rows[i]
		if row.Object.Object != nil {
			fmt.Fprintf(r.Command.OutOrStderr(), "Found non-nil Object in table: %s",
				row.Object.Object.GetObjectKind().GroupVersionKind())
		} else {
			if name, ok := getStringCell(row.Cells, nameCol); ok {
				if !strings.Contains(name, r.name) {
					continue
				}
			}
			if revision, ok := getStringCell(row.Cells, revisionCol); ok {
				if r.revision != "" && r.revision != revision {
					continue
				}
			}
		}

		// Row matches
		filtered = append(filtered, *row)
	}
	table.Rows = filtered
	return table, nil
}
