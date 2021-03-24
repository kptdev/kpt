package errors

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
)

// Error is an implementation of the error interface used in the kpt
// codebase.
// It is based on the design in https://commandcenter.blogspot.com/2017/12/error-handling-in-upspin.html
type Error struct {
	Path pkg.UniquePath

	Op Op

	Kind Kind

	Err error
}

func pad(b *strings.Builder, str string) {
	if b.Len() == 0 {
		return
	}
	b.WriteString(str)
}

func (e *Error) Error() string {
	b := new(strings.Builder)

	if e.Op != "" {
		pad(b, ": ")
		b.WriteString(string(e.Op))
	}

	if e.Path != "" {
		pad(b, ": ")
		b.WriteString(e.Path.String())
	}

	if e.Kind != 0 {
		pad(b, ": ")
		b.WriteString(e.Kind.String())
	}

	if e.Err != nil {
		if wrappedErr, ok := e.Err.(*Error); ok {
			if !wrappedErr.Zero() {
				pad(b, "\n\t")
				b.WriteString(wrappedErr.Error())
			}
		} else {
			pad(b, ": ")
			b.WriteString(e.Err.Error())
		}
	}
	if b.Len() == 0 {
		return "no error"
	}
	return b.String()
}

func (e *Error) Zero() bool {
	return e.Op == "" && e.Path == "" && e.Kind == 0 && e.Err == nil
}

type Op string

type Kind int

const (
	Other        Kind = iota // Unclassified. Will not be printed.
	Exist                    // Item already exists.
	Internal                 // Internal error.
	InvalidParam             // Value is not valid.
	MissingParam             // Required value is missing or empty.
	Git                      // Errors from Git
)

func (k Kind) String() string {
	switch k {
	case Other:
		return "other error"
	case Exist:
		return "item already exist"
	case Internal:
		return "internal error"
	case InvalidParam:
		return "invalid parameter value"
	case MissingParam:
		return "missing parameter value"
	case Git:
		return "git error"
	}
	return "unknown kind"
}

func E(args ...interface{}) error {
	if len(args) == 0 {
		panic("errors.E must have at least one argument")
	}

	e := &Error{}
	for _, arg := range args {
		switch a := arg.(type) {
		case pkg.UniquePath:
			e.Path = a
		case Op:
			e.Op = a
		case Kind:
			e.Kind = a
		case *Error:
			cp := *a
			e.Err = &cp
		case error:
			e.Err = a
		case string:
			e.Err = fmt.Errorf(a)
		default:
			panic(fmt.Errorf("unknown type %T for value %v in call to error.E", a, a))
		}
	}

	wrappedErr, ok := e.Err.(*Error)
	if !ok {
		return e
	}

	if e.Path == wrappedErr.Path {
		wrappedErr.Path = ""
	}

	if e.Op == wrappedErr.Op {
		wrappedErr.Op = ""
	}

	if e.Kind == wrappedErr.Kind {
		wrappedErr.Kind = 0
	}

	return e
}
