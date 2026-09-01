// Copyright 2021 The kpt Authors
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

// Package printer defines utilities to display kpt CLI and Porch output.
package printer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"k8s.io/klog/v2"
)

// TruncateOutput defines should output be truncated
var TruncateOutput bool

const (
	packagePrefixFormat = "Package %q:"
	defaultLogDepth     = 2
)

// ContextualFields holds key-value metadata pairs (e.g. image, tag, package, requestID, user).
type ContextualFields map[string]string

// Printer defines capabilities to display content in kpt CLI and Porch.
type Printer interface {
	// Contextual scoping methods (returns a child Printer with updated key-value fields)
	WithField(key, value string) Printer
	WithFields(fields ContextualFields) Printer
	WithPackage(pkgName string) Printer
	WithFunction(image, tag string) Printer

	// Structured lifecycle events
	PrintRunning(fnRef string, resourceCount int)
	PrintPass(fnRef string, duration time.Duration)
	PrintFail(fnRef string, duration time.Duration, err error)
	PrintResult(severity, msg string, targetRef string)
	PrintSummary(executedFnCnt, pkgCnt int, totalTime time.Duration)

	// Legacy printing methods
	PrintPackage(pkg *pkg.Pkg, leadingNewline bool)
	Printf(format string, args ...any)
	OptPrintf(opt *Options, format string, args ...any)

	// Stream accessors
	OutStream() io.Writer
	ErrStream() io.Writer
}

// Options are optional options for printer
type Options struct {
	// PkgPath is the unique path to the package
	PkgPath kptfilev1.UniquePath
	// PkgDisplayPath is the display path for the package
	PkgDisplayPath kptfilev1.DisplayPath
	// PkgDisplayName is the display name of the package.
	// It takes precedence over PkgPath and PkgDisplayPath in most logging scenarios.
	PkgDisplayName string
}

// NewOpt returns a pointer to new options
func NewOpt() *Options {
	return &Options{}
}

// Path sets the package unique path in options
func (opt *Options) Path(p kptfilev1.UniquePath) *Options {
	opt.PkgPath = p
	return opt
}

// DisplayPath sets the package display path in options
func (opt *Options) DisplayPath(p kptfilev1.DisplayPath) *Options {
	opt.PkgDisplayPath = p
	return opt
}

// DisplayName sets the package display name in options
func (opt *Options) DisplayName(name string) *Options {
	opt.PkgDisplayName = name
	return opt
}

// New returns an instance of stream-based Printer for kpt CLI.
func New(outStream, errStream io.Writer) Printer {
	if outStream == nil {
		outStream = os.Stdout
	}
	if errStream == nil {
		errStream = os.Stderr
	}
	return &printer{
		outStream: outStream,
		errStream: errStream,
		fields:    make(ContextualFields),
	}
}

// NewKlogPrinter returns a Printer that writes logs using klog.InfofDepth.
func NewKlogPrinter() Printer {
	return &printer{
		outStream: os.Stdout,
		errStream: os.Stderr,
		logFn: func(depth int, format string, args ...any) {
			klog.InfofDepth(depth, format, args...)
		},
		fields: make(ContextualFields),
	}
}

// NewWithLogFunc returns a Printer that delegates log printing to a custom log function.
func NewWithLogFunc(logFn func(depth int, format string, args ...any)) Printer {
	return &printer{
		outStream: os.Stdout,
		errStream: os.Stderr,
		logFn:     logFn,
		fields:    make(ContextualFields),
	}
}

// printer implements Printer for kpt CLI and Porch.
type printer struct {
	mu        sync.RWMutex
	outStream io.Writer
	errStream io.Writer
	logFn     func(depth int, format string, args ...any)
	fields    ContextualFields
}

// clone creates a copy of the printer with cloned contextual fields.
func (pr *printer) clone() *printer {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	newFields := make(ContextualFields, len(pr.fields))
	for k, v := range pr.fields {
		newFields[k] = v
	}

	return &printer{
		outStream: pr.outStream,
		errStream: pr.errStream,
		logFn:     pr.logFn,
		fields:    newFields,
	}
}

// WithField returns a child Printer with the specified key-value field attached.
func (pr *printer) WithField(key, value string) Printer {
	child := pr.clone()
	if value != "" {
		child.fields[key] = value
	} else {
		delete(child.fields, key)
	}
	return child
}

// WithFields returns a child Printer with the provided key-value fields attached.
func (pr *printer) WithFields(fields ContextualFields) Printer {
	child := pr.clone()
	for k, v := range fields {
		if v != "" {
			child.fields[k] = v
		} else {
			delete(child.fields, k)
		}
	}
	return child
}

// WithPackage returns a child Printer with the package field attached.
func (pr *printer) WithPackage(pkgName string) Printer {
	return pr.WithField("package", pkgName)
}

// WithFunction returns a child Printer with image and tag fields attached.
func (pr *printer) WithFunction(image, tag string) Printer {
	p := pr.WithField("image", image)
	if tag != "" {
		p = p.WithField("tag", tag)
	}
	return p
}

// OutStream returns the StdOut stream.
func (pr *printer) OutStream() io.Writer {
	return pr.outStream
}

// ErrStream returns the StdErr stream.
func (pr *printer) ErrStream() io.Writer {
	return pr.errStream
}

// formatFields returns sorted, formatted key-value pairs (e.g. image="set-labels" tag="latest").
func (pr *printer) formatFields(extraFields ...ContextualFields) string {
	pr.mu.RLock()
	combined := make(ContextualFields, len(pr.fields))
	for k, v := range pr.fields {
		combined[k] = v
	}
	pr.mu.RUnlock()

	for _, ef := range extraFields {
		for k, v := range ef {
			if v != "" {
				combined[k] = v
			}
		}
	}

	if len(combined) == 0 {
		return ""
	}

	keys := make([]string, 0, len(combined))
	for k := range combined {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%s=%s", k, strconv.Quote(combined[k])))
	}
	return sb.String()
}

func (pr *printer) printInternal(format string, args ...any) {
	if pr.logFn != nil {
		pr.logFn(defaultLogDepth+1, format, args...)
	} else {
		fmt.Fprintf(pr.errStream, format, args...)
	}
}

// PrintRunning outputs a [RUNNING] lifecycle event.
func (pr *printer) PrintRunning(fnRef string, resourceCount int) {
	extra := ContextualFields{}
	if resourceCount > 0 {
		extra["resourceCount"] = strconv.Itoa(resourceCount)
	}
	attrStr := pr.formatFields(extra)

	if attrStr != "" {
		pr.printInternal("[RUNNING] %s\n", attrStr)
	} else if resourceCount > 0 {
		pr.printInternal("[RUNNING] %s on %d resource(s)\n", strconv.Quote(fnRef), resourceCount)
	} else {
		pr.printInternal("[RUNNING] %s\n", strconv.Quote(fnRef))
	}
}

// PrintPass outputs a [PASS] lifecycle event.
func (pr *printer) PrintPass(fnRef string, duration time.Duration) {
	extra := ContextualFields{}
	if duration > 0 {
		extra["time"] = duration.Truncate(time.Millisecond).String()
	}
	attrStr := pr.formatFields(extra)

	if attrStr != "" {
		pr.printInternal("[PASS] %s\n", attrStr)
	} else {
		pr.printInternal("[PASS] %q in %v\n", fnRef, duration.Truncate(time.Millisecond))
	}
}

// PrintFail outputs a [FAIL] lifecycle event.
func (pr *printer) PrintFail(fnRef string, duration time.Duration, err error) {
	extra := ContextualFields{}
	if duration > 0 {
		extra["time"] = duration.Truncate(time.Millisecond).String()
	}
	if err != nil {
		extra["error"] = err.Error()
	}
	attrStr := pr.formatFields(extra)

	if attrStr != "" {
		pr.printInternal("[FAIL] %s\n", attrStr)
	} else {
		pr.printInternal("[FAIL] %q in %v\n", fnRef, duration.Truncate(time.Millisecond))
	}
}

// PrintResult outputs a structured result item line.
func (pr *printer) PrintResult(severity, msg string, targetRef string) {
	if targetRef != "" {
		pr.printInternal("    [%s] %s: %s\n", severity, targetRef, msg)
	} else {
		pr.printInternal("    [%s]: %s\n", severity, msg)
	}
}

// PrintSummary outputs a pipeline execution summary line.
func (pr *printer) PrintSummary(executedFnCnt, pkgCnt int, totalTime time.Duration) {
	extra := ContextualFields{}
	if totalTime > 0 {
		extra["time"] = totalTime.Truncate(time.Millisecond).String()
	}
	attrStr := pr.formatFields(extra)

	if attrStr != "" {
		pr.printInternal("Successfully executed %d function(s) in %d package(s) %s\n", executedFnCnt, pkgCnt, attrStr)
	} else {
		pr.printInternal("Successfully executed %d function(s) in %d package(s).\n", executedFnCnt, pkgCnt)
	}
}

// PrintPackage prints the package display path.
func (pr *printer) PrintPackage(p *pkg.Pkg, leadingNewline bool) {
	if leadingNewline && pr.logFn == nil {
		fmt.Fprint(pr.errStream, "\n")
	}
	if pr.logFn != nil {
		pr.logFn(defaultLogDepth+1, packagePrefixFormat, p.DisplayPath)
	} else {
		fmt.Fprintf(pr.errStream, "Package %q:\n", p.DisplayPath)
	}
}

// Printf is the wrapper over fmt.Printf that displays the output.
func (pr *printer) Printf(format string, args ...any) {
	pr.printInternal(format, args...)
}

// OptPrintf is the wrapper over fmt.Printf that displays output according to the options.
func (pr *printer) OptPrintf(opt *Options, format string, args ...any) {
	if opt == nil {
		pr.printInternal(format, args...)
		return
	}

	var prefix string
	switch {
	case opt.PkgDisplayName != "":
		prefix = fmt.Sprintf(packagePrefixFormat, opt.PkgDisplayName)
	case !opt.PkgDisplayPath.Empty():
		prefix = fmt.Sprintf(packagePrefixFormat, string(opt.PkgDisplayPath))
	case !opt.PkgPath.Empty():
		relPath, err := opt.PkgPath.RelativePath()
		if err != nil {
			relPath = string(opt.PkgPath)
		}
		prefix = fmt.Sprintf(packagePrefixFormat, relPath)
	}

	pr.printInternal(prefix+format, args...)
}

// Context keys and helper functions

type contextKey int

const printerKey contextKey = 0

// FromContextOrDie returns the Printer instance associated with the context.
func FromContextOrDie(ctx context.Context) Printer {
	pr, ok := ctx.Value(printerKey).(Printer)
	if ok {
		return pr
	}
	panic("printer missing in context")
}

// WithContext creates a new context setting the printer instance.
func WithContext(ctx context.Context, pr Printer) context.Context {
	return context.WithValue(ctx, printerKey, pr)
}

