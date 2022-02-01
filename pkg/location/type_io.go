package location

import (
	"fmt"
	"io"
)

type InputStream struct {
	Reader io.Reader
}

var _ Reference = InputStream{}

func stdinParser(value string, opt options) (Reference, error) {
	if value == "-" {
		return InputStream{
			Reader: opt.stdin,
		}, nil
	}
	return nil, nil
}

// String implements location.Reference
func (ref InputStream) String() string {
	return fmt.Sprintf("type:io reader:%v", ref.Reader)
}

// Type implements location.Reference
func (ref InputStream) Type() string {
	return "io"
}

// Validate implements location.Reference
func (ref InputStream) Validate() error {
	return nil
}

type OutputStream struct {
	Writer io.Writer
}

var _ Reference = OutputStream{}

func stdoutParser(value string, opt options) (Reference, error) {
	if value == "-" {
		return OutputStream{
			Writer: opt.stdout,
		}, nil
	}
	return nil, nil
}

// String implements location.Reference
func (ref OutputStream) String() string {
	return fmt.Sprintf("type:io writer:%v", ref.Writer)
}

// Type implements location.Reference
func (ref OutputStream) Type() string {
	return "io"
}

// Validate implements location.Reference
func (ref OutputStream) Validate() error {
	return nil
}
