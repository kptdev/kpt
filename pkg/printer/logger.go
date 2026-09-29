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
	"log/slog"
	"strings"

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

type StdLogger struct{}

func (*StdLogger) PrintM(msg string, args map[string]any) {
	sb := strings.Builder{}
	sb.WriteString(msg)

	for k, v := range args {
		_, _ = fmt.Fprintf(&sb, " %s=%v", k, v)
	}

	fmt.Println(sb.String())
}

type KlogLogger struct{}

func (*KlogLogger) PrintM(msg string, args map[string]any) {
	klog.InfoS(msg, fromMap(args)...)
}

type SlogLogger struct{}

func (sl *SlogLogger) PrintM(msg string, args map[string]any) {
	slog.Info(msg, fromMap(args)...)
}

func fromMap(args map[string]any) []any {
	output := make([]any, 0, len(args))
	for k, v := range args {
		output = append(output, k, v)
	}
	return output
}
