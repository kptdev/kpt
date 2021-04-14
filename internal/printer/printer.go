// Package printer defines utilities to display kpt CLI output.
package printer

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/GoogleContainerTools/kpt/internal/types"
)

// Printer defines capabilities to display content in kpt CLI.
// The main intention, at the moment, is to abstract away printing
// output in the CLI so that we can evolve the kpt CLI UX.
type Printer interface {
	PkgPrintf(pkgPath types.UniquePath, format string, args ...interface{})
	Printf(format string, args ...interface{})
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

// PkgPrintf accepts an optional pkgpath to display the message with
// package context.
func (pr *printer) PkgPrintf(pkgPath types.UniquePath, format string, args ...interface{}) {
	// try to print relative path of the pkg if we can else use abs path
	relPath, err := pkgPath.RelativePath()
	if err != nil {
		relPath = string(pkgPath)
	}
	if string(pkgPath) != "" {
		format = fmt.Sprintf("package %q: ", relPath) + format
	}
	fmt.Fprintf(pr.outStream, format, args...)
}

// The key type is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey int

// printerKey is the context key for the printer.  Its value of zero is
// arbitrary.  If this package defined other context keys, they would have
// different integer values.
const printerKey contextKey = 0

// Printf is the wrapper over fmt.Printf that displays the output to
// the configured output writer.
func (pr *printer) Printf(format string, args ...interface{}) {
	fmt.Fprintf(pr.outStream, format, args...)
}

// Helper functions to set and retrieve printer instance from a context.
// Defining them here avoids the context key collision.

// FromContext returns printer instance associated with the context.
func FromContextOrDie(ctx context.Context) Printer {
	pr, ok := ctx.Value(printerKey).(*printer)
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
