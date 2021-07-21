// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"time"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/integer"
)

// ColumnDef is an implementation of the ColumnDefinition interface.
// It can be used to define simple columns that doesn't need additional
// knowledge about the actual type of the provided Resource besides the
// information available through the interface.
type ColumnDef struct {
	ColumnName        string
	ColumnHeader      string
	ColumnWidth       int
	PrintResourceFunc func(w io.Writer, width int, r Resource) (int, error)
}

// Name returns the name of the column.
func (c ColumnDef) Name() string {
	return c.ColumnName
}

// Header returns the header that should be printed for
// the column.
func (c ColumnDef) Header() string {
	return c.ColumnHeader
}

// Width returns the width of the column.
func (c ColumnDef) Width() int {
	return c.ColumnWidth
}

// PrintResource is called by the BaseTablePrinter to output the
// content of a particular column. This implementation just delegates
// to the provided PrintResourceFunc.
func (c ColumnDef) PrintResource(w io.Writer, width int, r Resource) (int, error) {
	return c.PrintResourceFunc(w, width, r)
}

// MustColumn returns the pre-defined column definition with the
// provided name. If the name doesn't exist, it will panic.
func MustColumn(name string) ColumnDef {
	c, found := columnDefinitions[name]
	if !found {
		panic(fmt.Errorf("unknown column name %q", name))
	}
	return c
}

var (
	columnDefinitions = map[string]ColumnDef{
		// namespace defines a column that output the namespace of the
		// resource, or nothing in the case of clusterscoped resources.
		"namespace": {
			ColumnName:   "namespace",
			ColumnHeader: "NAMESPACE",
			ColumnWidth:  10,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (int,
				error) {
				namespace := r.Identifier().Namespace
				if len(namespace) > width {
					namespace = namespace[:width]
				}
				_, err := fmt.Fprint(w, namespace)
				return len(namespace), err
			},
		},
		// resource defines a column that outputs the kind and name of a
		// resource.
		"resource": {
			ColumnName:   "resource",
			ColumnHeader: "RESOURCE",
			ColumnWidth:  40,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (int,
				error) {
				text := fmt.Sprintf("%s/%s", r.Identifier().GroupKind.Kind,
					r.Identifier().Name)
				if len(text) > width {
					text = text[:width]
				}
				_, err := fmt.Fprint(w, text)
				return len(text), err
			},
		},
		// status defines a column that outputs the status of a resource. It
		// will use ansii escape codes to color the output.
		"status": {
			ColumnName:   "status",
			ColumnHeader: "STATUS",
			ColumnWidth:  10,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (int,
				error) {
				rs := r.ResourceStatus()
				if rs == nil {
					return 0, nil
				}
				s := rs.Status.String()
				if len(s) > width {
					s = s[:width]
				}
				color, setColor := common.ColorForStatus(rs.Status)
				var outputStatus string
				if setColor {
					outputStatus = common.SprintfWithColor(color, s)
				} else {
					outputStatus = s
				}
				_, err := fmt.Fprint(w, outputStatus)
				return len(s), err
			},
		},
		// conditions defines a column that outputs the conditions for
		// a resource. The output will be in colors.
		"conditions": {
			ColumnName:   "conditions",
			ColumnHeader: "CONDITIONS",
			ColumnWidth:  40,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (int,
				error) {
				rs := r.ResourceStatus()
				if rs == nil {
					return 0, nil
				}
				u := rs.Resource
				if u == nil {
					return fmt.Fprintf(w, "-")
				}

				conditions, found, err := unstructured.NestedSlice(u.Object,
					"status", "conditions")
				if !found || err != nil || len(conditions) == 0 {
					return fmt.Fprintf(w, "<None>")
				}

				realLength := 0
				for i, cond := range conditions {
					condition := cond.(map[string]interface{})
					conditionType := condition["type"].(string)
					conditionStatus := condition["status"].(string)
					var color common.Color
					switch conditionStatus {
					case "True":
						color = common.GREEN
					case "False":
						color = common.RED
					default:
						color = common.YELLOW
					}
					remainingWidth := width - realLength
					if len(conditionType) > remainingWidth {
						conditionType = conditionType[:remainingWidth]
					}
					_, err := fmt.Fprint(w, common.SprintfWithColor(color, conditionType))
					if err != nil {
						return realLength, err
					}
					realLength += len(conditionType)
					if i < len(conditions)-1 && width-realLength > 2 {
						_, err = fmt.Fprintf(w, ",")
						if err != nil {
							return realLength, err
						}
						realLength += 1
					}
				}
				return realLength, nil
			},
		},
		// age defines a column that outputs the age of a resource computed
		// by looking at the creationTimestamp field.
		"age": {
			ColumnName:   "age",
			ColumnHeader: "AGE",
			ColumnWidth:  6,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (i int, err error) {
				rs := r.ResourceStatus()
				if rs == nil {
					return 0, nil
				}
				u := rs.Resource
				if u == nil {
					return fmt.Fprint(w, "-")
				}

				timestamp, found, err := unstructured.NestedString(u.Object,
					"metadata", "creationTimestamp")
				if !found || err != nil || timestamp == "" {
					return fmt.Fprint(w, "-")
				}
				parsedTime, err := time.Parse(time.RFC3339, timestamp)
				if err != nil {
					return fmt.Fprint(w, "-")
				}
				age := time.Since(parsedTime)
				switch {
				case age.Seconds() <= 90:
					return fmt.Fprintf(w, "%ds",
						integer.RoundToInt32(age.Round(time.Second).Seconds()))
				case age.Minutes() <= 90:
					return fmt.Fprintf(w, "%dm",
						integer.RoundToInt32(age.Round(time.Minute).Minutes()))
				default:
					return fmt.Fprintf(w, "%dh",
						integer.RoundToInt32(age.Round(time.Hour).Hours()))
				}
			},
		},
		// message defines a column that outputs the message from a
		// ResourceStatus, or if there is a non-nil error, output the text
		// from the error instead.
		"message": {
			ColumnName:   "message",
			ColumnHeader: "MESSAGE",
			ColumnWidth:  40,
			PrintResourceFunc: func(w io.Writer, width int, r Resource) (i int, err error) {
				rs := r.ResourceStatus()
				if rs == nil {
					return 0, nil
				}
				var message string
				if rs.Error != nil {
					message = rs.Error.Error()
				} else {
					message = rs.Message
				}
				if len(message) > width {
					message = message[:width]
				}
				return fmt.Fprint(w, message)
			},
		},
	}
)
