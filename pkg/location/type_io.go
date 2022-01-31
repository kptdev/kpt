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

func (ref InputStream) String() string {
	return fmt.Sprintf("type:io reader:%v", ref.Reader)
}

func (ref InputStream) Type() string {
	return "io"
}

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

func (ref OutputStream) String() string {
	return fmt.Sprintf("type:io writer:%v", ref.Writer)
}

func (ref OutputStream) Type() string {
	return "io"
}

func (ref OutputStream) Validate() error {
	return nil
}
