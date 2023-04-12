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

// Package printer defines utilities to display kpt CLI output.
package printer

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
)

// TruncateOutput defines should output be truncated
var TruncateOutput bool

// Printer defines capabilities to display content in kpt CLI.
// The main intention, at the moment, is to abstract away printing
// output in the CLI so that we can evolve the kpt CLI UX.
type Printer interface {
	PrintPackage(pkg *pkg.Pkg, leadingNewline bool)
	Printf(format string, args ...interface{})
	OptPrintf(opt *Options, format string, args ...interface{})
	OutStream() io.Writer
	ErrStream() io.Writer
}

// Options are optional options for printer
type Options struct {
	// PkgPath is the unique path to the package
	PkgPath types.UniquePath
	// PkgDisplayPath is the display path for the package
	PkgDisplayPath types.DisplayPath
}

// NewOpt returns a pointer to new options
func NewOpt() *Options {
	return &Options{}
}

// Pkg sets the package unique path in options
func (opt *Options) Pkg(p types.UniquePath) *Options {
	opt.PkgPath = p
	return opt
}

// PkgDisplayPath sets the package display path in options
func (opt *Options) PkgDisplay(p types.DisplayPath) *Options {
	opt.PkgDisplayPath = p
	return opt
}

// New returns an instance of Printer.
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
	}
}

// printer implements default Printer to be used in kpt codebase.
type printer struct {
	outStream io.Writer
	errStream io.Writer
}

// The key type is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey int

// printerKey is the context key for the printer.  Its value of zero is
// arbitrary.  If this package defined other context keys, they would have
// different integer values.
const printerKey contextKey = 0

// OutStream returns the StdOut stream, this can be used by callers to print
// command output to stdout, do not print error/debug logs to this stream
func (pr *printer) OutStream() io.Writer {
	return pr.outStream
}

// ErrStream returns the StdErr stream, this can be used by callers to print
// command output to stderr, print only error/debug/info logs to this stream
func (pr *printer) ErrStream() io.Writer {
	return pr.errStream
}

// PrintPackage prints the package display path to stderr
func (pr *printer) PrintPackage(p *pkg.Pkg, leadingNewline bool) {
	if leadingNewline {
		fmt.Fprint(pr.errStream, "\n")
	}
	fmt.Fprintf(pr.errStream, "Package %q:\n", p.DisplayPath)
}

// Printf is the wrapper over fmt.Printf that displays the output.
// this will print messages to stderr stream
func (pr *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(pr.errStream, format, args...)
}

// OptPrintf is the wrapper over fmt.Printf that displays the output according
// to the opt, this will print messages to stderr stream
// https://mehulkar.com/blog/2017/11/stdout-vs-stderr/
func (pr *printer) OptPrintf(opt *Options, format string, args ...interface{}) {
	if opt == nil {
		fmt.Fprintf(pr.errStream, format, args...)
		return
	}
	o := pr.errStream
	if !opt.PkgDisplayPath.Empty() {
		format = fmt.Sprintf("Package %q: ", string(opt.PkgDisplayPath)) + format
	} else if !opt.PkgPath.Empty() {
		// try to print relative path of the pkg if we can else use abs path
		relPath, err := opt.PkgPath.RelativePath()
		if err != nil {
			relPath = string(opt.PkgPath)
		}
		format = fmt.Sprintf("Package %q: ", relPath) + format
	}
	fmt.Fprintf(o, format, args...)
}

// Helper functions to set and retrieve printer instance from a context.
// Defining them here avoids the context key collision.

// FromContext returns printer instance associated with the context.
func FromContextOrDie(ctx context.Context) Printer {
	pr, ok := ctx.Value(printerKey).(Printer)
	if ok {
		return pr
	}
	panic("printer missing in context")
}

// WithContext creates new context from the given parent context
// by setting the printer instance.
func WithContext(ctx context.Context, pr Printer) context.Context {
	return context.WithValue(ctx, printerKey, pr)
}
