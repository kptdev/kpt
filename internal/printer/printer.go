// Copyright 2021 Google LLC
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
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/types"
)

const (
	// FnIndentation is the number of spaces at the beginning of each line of
	// function running progress.
	FnIndentation = 0
)

// DisableOutputTruncate defines should output be truncated
var DisableOutputTruncate bool

// Printer defines capabilities to display content in kpt CLI.
// The main intention, at the moment, is to abstract away printing
// output in the CLI so that we can evolve the kpt CLI UX.
type Printer interface {
	Printf(format string, args ...interface{})
	OptPrintf(opt *Options, format string, args ...interface{})
}

// Options are optional options for printer
type Options struct {
	// Indetation is the number of spaces added at the beginning
	// of each line
	Indentation int
	// OutputToStderr indicates should output be printed to stderr instead
	// of stdout
	OutputToStderr bool
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

// Indent sets the output indentation in options
func (opt *Options) Indent(i int) *Options {
	opt.Indentation = i
	return opt
}

// Stderr sets output to stderr in options
func (opt *Options) Stderr() *Options {
	opt.OutputToStderr = true
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

// Printf is the wrapper over fmt.Printf that displays the output.
func (pr *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(pr.outStream, format, args...)
}

// OptPrintf is the wrapper over fmt.Printf that displays the output according
// to the opt.
func (pr *printer) OptPrintf(opt *Options, format string, args ...interface{}) {
	if opt == nil {
		fmt.Fprintf(pr.outStream, format, args...)
		return
	}
	o := pr.outStream
	if opt.OutputToStderr {
		o = pr.errStream
	}
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
	if opt.Indentation != 0 {
		// we need to add indentation to each line
		indentPrintf(o, opt.Indentation, format, args...)
		return
	}
	fmt.Fprintf(o, format, args...)
}

func indentPrintf(w io.Writer, indentation int, format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		// don't add newline for last line to respect the original input
		// format
		newline := "\n"
		if i == len(lines)-1 {
			newline = ""
		}
		if l == "" {
			// don't print indentation when the line is empty
			fmt.Fprint(w, newline)
		} else {
			fmt.Fprint(w, strings.Repeat(" ", indentation)+l+newline)
		}
	}
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
