package location

import (
	"fmt"
	"io"
)

type InputStream struct {
	Reader io.Reader
}

var _ Reference = InputStream{}

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

func (ref OutputStream) String() string {
	return fmt.Sprintf("type:io writer:%v", ref.Writer)
}

func (ref OutputStream) Type() string {
	return "io"
}

func (ref OutputStream) Validate() error {
	return nil
}

type InputOutputStream struct {
	Reader io.Reader
	Writer io.Writer
}

var _ Reference = InputOutputStream{}

func (ref InputOutputStream) String() string {
	return fmt.Sprintf("type:io reader:%v writer:%v", ref.Reader, ref.Writer)
}

func (ref InputOutputStream) Type() string {
	return "io"
}

func (ref InputOutputStream) Validate() error {
	return nil
}
