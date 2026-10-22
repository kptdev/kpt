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

package printer

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"

	"k8s.io/klog/v2"
)

type MapPrinter interface {
	PrintM(msg string, args map[string]any)
}

type StructuredPrinter interface {
	PrintS(msg string, args ...any)
}

type adapter struct {
	inner StructuredPrinter
}

var _ MapPrinter = &adapter{}

func (p *adapter) PrintM(msg string, args map[string]any) {
	p.inner.PrintS(msg, fromMap(args)...)
}

func FromStructured(p StructuredPrinter) MapPrinter {
	return &adapter{p}
}

// IOLogger writes to Writer if set and os.Stdout if not
type IOLogger struct {
	Writer io.Writer
}

func (l *IOLogger) PrintM(msg string, args map[string]any) {
	if l.Writer == nil {
		l.Writer = os.Stdout
	}
	_, _ = fmt.Fprintf(l.Writer, msg)

	for _, k := range slices.Sorted(maps.Keys(args)) {
		_, _ = fmt.Fprintf(l.Writer, " %s=%v", k, args[k])
	}
}

type KlogLogger struct{}

func (*KlogLogger) PrintM(msg string, args map[string]any) {
	klog.InfoS(msg, fromMap(args)...)
}

type SlogLogger struct{}

func (sl *SlogLogger) PrintM(msg string, args map[string]any) {
	slog.Info(msg, fromMap(args)...)
}

// fromMap returns a sorted, structured-log-compatible array for a map of arguments
func fromMap(args map[string]any) []any {
	output := make([]any, 0, len(args))
	for _, k := range slices.Sorted(maps.Keys(args)) {
		output = append(output, k, args[k])
	}
	return output
}
