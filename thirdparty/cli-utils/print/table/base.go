// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/common"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// ColumnDefinition defines the columns that should be printed.
type ColumnDefinition interface {
	Name() string
	Header() string
	Width() int
	PrintResource(w io.Writer, width int, r Resource) (int, error)
}

// ResourceStates defines the interface that must be implemented
// by the object that provides information about the resources
// that should be printed.
type ResourceStates interface {
	Resources() []Resource
	Error() error
}

// Resource defines the interface that each of the Resource
// objects must implement.
type Resource interface {
	Identifier() object.ObjMetadata
	ResourceStatus() *pe.ResourceStatus
	SubResources() []Resource
}

// BaseTablePrinter provides functionality for printing information
// about a set of resources into a table format.
// The printer will print to the Out stream defined in IOStreams,
// and will print into the format defined by the Column definitions.
type BaseTablePrinter struct {
	IOStreams genericclioptions.IOStreams
	Columns   []ColumnDefinition
}

// PrintTable prints the resources defined in ResourceStates. It will
// print subresources if they exist.
// moveUpCount defines how many lines the printer should move up
// before starting printing. The return value is how many lines
// were printed.
func (t *BaseTablePrinter) PrintTable(rs ResourceStates,
	moveUpCount int) int {
	for i := 0; i < moveUpCount; i++ {
		t.moveUp()
		t.eraseCurrentLine()
	}

	linePrintCount := 0
	for i, column := range t.Columns {
		format := fmt.Sprintf("%%-%ds", column.Width())
		t.printOrDie(format, column.Header())
		if i == len(t.Columns)-1 {
			t.printOrDie("\n")
			linePrintCount++
		} else {
			t.printOrDie("  ")
		}
	}

	for _, resource := range rs.Resources() {
		for i, column := range t.Columns {
			written, err := column.PrintResource(t.IOStreams.Out, column.Width(), resource)
			if err != nil {
				panic(err)
			}
			remainingSpace := column.Width() - written
			t.printOrDie(strings.Repeat(" ", remainingSpace))
			if i == len(t.Columns)-1 {
				t.printOrDie("\n")
				linePrintCount++
			} else {
				t.printOrDie("  ")
			}
		}

		linePrintCount += t.printSubTable(resource.SubResources(), "")
	}

	return linePrintCount
}

// printSubTable prints out any subresources that belong to the
// top-level resources. This function takes care of printing the correct tree
// structure and indentation.
func (t *BaseTablePrinter) printSubTable(resources []Resource,
	prefix string) int {
	linePrintCount := 0
	for j, resource := range resources {
		for i, column := range t.Columns {
			availableWidth := column.Width()
			if column.Name() == "resource" {
				if j < len(resources)-1 {
					t.printOrDie(prefix + `├─ `)
				} else {
					t.printOrDie(prefix + `└─ `)
				}
				availableWidth -= utf8.RuneCountInString(prefix) + 3
			}
			written, err := column.PrintResource(t.IOStreams.Out,
				availableWidth, resource)
			if err != nil {
				panic(err)
			}
			remainingSpace := availableWidth - written
			t.printOrDie(strings.Repeat(" ", remainingSpace))
			if i == len(t.Columns)-1 {
				t.printOrDie("\n")
				linePrintCount++
			} else {
				t.printOrDie("  ")
			}
		}

		var prefix string
		if j < len(resources)-1 {
			prefix = `│  `
		} else {
			prefix = "   "
		}
		linePrintCount += t.printSubTable(resource.SubResources(), prefix)
	}
	return linePrintCount
}

func (t *BaseTablePrinter) printOrDie(format string, a ...interface{}) {
	_, err := fmt.Fprintf(t.IOStreams.Out, format, a...)
	if err != nil {
		panic(err)
	}
}

func (t *BaseTablePrinter) moveUp() {
	t.printOrDie("%c[%dA", common.ESC, 1)
}

func (t *BaseTablePrinter) eraseCurrentLine() {
	t.printOrDie("%c[2K\r", common.ESC)
}
