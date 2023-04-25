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

package get

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/docs/generated/rpkgdocs"
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/options"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/kubectl/pkg/cmd/get"
)

const (
	command = "cmdrpkgget"
)

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:        ctx,
		getFlags:   options.Get{ConfigFlags: rcg},
		printFlags: get.NewGetPrintFlags(),
	}
	cmd := &cobra.Command{
		Use:        "get",
		Aliases:    []string{"list"},
		SuggestFor: []string{},
		Short:      rpkgdocs.GetShort,
		Long:       rpkgdocs.GetShort + "\n" + rpkgdocs.GetLong,
		Example:    rpkgdocs.GetExamples,
		PreRunE:    r.preRunE,
		RunE:       r.runE,
		Hidden:     porch.HidePorchCommands,
	}
	r.Command = cmd

	// Create flags
	cmd.Flags().StringVar(&r.packageName, "name", "", "Name of the packages to get. Any package whose name contains this value will be included in the results.")
	cmd.Flags().StringVar(&r.revision, "revision", "", "Revision of the packages to get. Any package whose revision matches this value will be included in the results.")
	cmd.Flags().StringVar(&r.workspace, "workspace", "",
		"WorkspaceName of the packages to get. Any package whose workspaceName matches this value will be included in the results.")

	r.getFlags.AddFlags(cmd)
	r.printFlags.AddFlags(cmd)
	return r
}

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

type runner struct {
	ctx      context.Context
	getFlags options.Get
	Command  *cobra.Command

	// Flags
	packageName string
	revision    string
	workspace   string
	printFlags  *get.PrintFlags

	requestTable bool
}

func (r *runner) preRunE(cmd *cobra.Command, _ []string) error {
	// Print the namespace if we're spanning namespaces
	if r.getFlags.AllNamespaces {
		r.printFlags.HumanReadableFlags.WithNamespace = true
	}

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
	b, err := r.getFlags.ResourceBuilder()
	if err != nil {
		return err
	}

	if r.requestTable {
		scheme := runtime.NewScheme()
		// Accept PartialObjectMetadata and Table
		if err := metav1.AddMetaToScheme(scheme); err != nil {
			return fmt.Errorf("error building runtime.Scheme: %w", err)
		}
		b = b.WithScheme(scheme, schema.GroupVersion{Version: "v1"})
	} else {
		// We want to print the server version, not whatever version we happen to have compiled in
		b = b.Unstructured()
	}

	useSelectors := true
	if len(args) > 0 {
		b = b.ResourceNames("packagerevisions", args...)
		// We can't pass selectors here, get an error "Error: selectors and the all flag cannot be used when passing resource/name arguments"
		// TODO: cli-utils bug?  I think there is a metadata.name field selector (used for single object watch)
		useSelectors = false
	} else {
		b = b.ResourceTypes("packagerevisions")
	}

	if useSelectors {
		fieldSelector := fields.Everything()
		if r.revision != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.revision", r.revision)
		}
		if r.workspace != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.workspaceName", r.workspace)
		}
		if r.packageName != "" {
			fieldSelector = fields.OneTermEqualSelector("spec.packageName", r.packageName)
		}
		if s := fieldSelector.String(); s != "" {
			b = b.FieldSelectorParam(s)
		} else {
			b = b.SelectAllParam(true)
		}
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

	// Decode json objects in tables (likely PartialObjectMetadata)
	for _, i := range infos {
		if table, ok := i.Object.(*metav1.Table); ok {
			for i := range table.Rows {
				row := &table.Rows[i]
				if row.Object.Object == nil && row.Object.Raw != nil {
					u := &unstructured.Unstructured{}
					if err := u.UnmarshalJSON(row.Object.Raw); err != nil {
						klog.Warningf("error parsing raw object: %v", err)
					}
					row.Object.Object = u
				}
			}
		}
	}

	// Apply any filters we couldn't pass down as field selectors
	for _, i := range infos {
		switch obj := i.Object.(type) {
		case *unstructured.Unstructured:
			match, err := r.packageRevisionMatches(obj)
			if err != nil {
				return errors.E(op, err)
			}
			if match {
				objs = append(objs, obj)
			}
		case *metav1.Table:
			// Technically we should have applied this as a field-selector, so this might not be necessary
			if err := r.filterTableRows(obj); err != nil {
				return err
			}
			objs = append(objs, obj)
		default:
			return errors.E(op, fmt.Sprintf("Unrecognized response %T", obj))
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

func (r *runner) packageRevisionMatches(o *unstructured.Unstructured) (bool, error) {
	packageName, _, err := unstructured.NestedString(o.Object, "spec", "packageName")
	if err != nil {
		return false, err
	}
	revision, _, err := unstructured.NestedString(o.Object, "spec", "revision")
	if err != nil {
		return false, err
	}
	workspace, _, err := unstructured.NestedString(o.Object, "spec", "workspaceName")
	if err != nil {
		return false, err
	}
	if r.packageName != "" && r.packageName != packageName {
		return false, nil
	}
	if r.revision != "" && r.revision != revision {
		return false, nil
	}
	if r.workspace != "" && r.workspace != workspace {
		return false, nil
	}
	return true, nil
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

func (r *runner) filterTableRows(table *metav1.Table) error {
	filtered := make([]metav1.TableRow, 0, len(table.Rows))
	packageNameCol := findColumn(table.ColumnDefinitions, "Package")
	revisionCol := findColumn(table.ColumnDefinitions, "Revision")
	workspaceCol := findColumn(table.ColumnDefinitions, "WorkspaceName")

	for i := range table.Rows {
		row := &table.Rows[i]

		if packageName, ok := getStringCell(row.Cells, packageNameCol); ok {
			if r.packageName != "" && r.packageName != packageName {
				continue
			}
		}
		if revision, ok := getStringCell(row.Cells, revisionCol); ok {
			if r.revision != "" && r.revision != revision {
				continue
			}
		}
		if workspace, ok := getStringCell(row.Cells, workspaceCol); ok {
			if r.workspace != "" && r.workspace != workspace {
				continue
			}
		}

		// Row matches
		filtered = append(filtered, *row)
	}
	table.Rows = filtered
	return nil
}
