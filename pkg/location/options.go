package location

import (
	"context"
	"io"
)

type options struct {
	ctx    context.Context
	stdin  io.Reader
	stdout io.Writer
}

func makeOptions(opts ...Option) options {
	opt := options{}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// Option is a functional option for location parsing.
type Option func(*options)

// WithDefaultTag sets the default tag that will be used if one is not provided.
func WithContext(ctx context.Context) Option {
	return func(opts *options) {
		opts.ctx = ctx
	}
}

// WithStdin enables parser to assign "-" location onto an input io.Reader
func WithStdin(r io.Reader) Option {
	return func(opts *options) {
		opts.stdin = r
	}
}

// WithStdout enables parser to assign "-" location onto an output io.Writer
func WithStdout(w io.Writer) Option {
	return func(opts *options) {
		opts.stdout = w
	}
}
