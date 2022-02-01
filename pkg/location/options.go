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

package location

import (
	"context"
	"io"
)

type options struct {
	ctx     context.Context
	stdin   io.Reader
	stdout  io.Writer
	parsers []parser
}

func makeOptions(opts ...Option) options {
	opt := options{}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

type parser func(value string, opt options) (Reference, error)

// Option is a functional option for location parsing.
type Option func(*options)

// WithDefaultTag sets the default tag that will be used if one is not provided.
func WithContext(ctx context.Context) Option {
	return func(opts *options) {
		opts.ctx = ctx
	}
}

// WithStdin enables parser to assign "-" location onto an input io.Reader
func WithStdin(reader io.Reader) Option {
	return func(opts *options) {
		opts.stdin = reader
		opts.parsers = append(opts.parsers, stdinParser)
	}
}

// WithStdout enables parser to assign "-" location onto an output io.Writer
func WithStdout(writer io.Writer) Option {
	return func(opts *options) {
		opts.stdout = writer
		opts.parsers = append(opts.parsers, stdoutParser)
	}
}

// WithGit enables standard parsing for the location.Git Reference type
func WithGit() Option {
	return func(opts *options) {
		opts.parsers = append(opts.parsers, parseGit)
	}
}

// WithOci enables standard parsing for the location.Oci Reference type
func WithOci() Option {
	return func(opts *options) {
		opts.parsers = append(opts.parsers, parseOci)
	}
}

// WithDir enables standard parsing for the location.Dir Reference type
func WithDir() Option {
	return func(opts *options) {
		opts.parsers = append(opts.parsers, parseDir)
	}
}

// WithParser enables parsing custom Reference type
func WithParser(parser func(value string) (Reference, error)) Option {
	return func(opts *options) {
		opts.parsers = append(opts.parsers, func(value string, opt options) (Reference, error) {
			return parser(value)
		})
	}
}
