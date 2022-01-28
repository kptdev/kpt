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

type OutputStream struct {
	Writer io.Writer
}

var _ Reference = OutputStream{}

func (ref OutputStream) String() string {
	return fmt.Sprintf("type:io writer:%v", ref.Writer)
}

type InputOutputStream struct {
	Reader io.Reader
	Writer io.Writer
}

var _ Reference = InputOutputStream{}

func (ref InputOutputStream) String() string {
	return fmt.Sprintf("type:io reader:%v writer:%v", ref.Reader, ref.Writer)
}
