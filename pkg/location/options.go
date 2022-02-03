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
	Options(opts...)(&opt)
	return opt
}

type parser func(value string, opt options) result

// result is returned by each parser, called in order.
//
// `ref` is assigned if the parser successfully recognized and processed the location string.
//
// `err` is assigned if the parser unambiguously recognized the format string, which then failed to parse.
// An example is "oci://imagename:tag+name" because oci:// is unambiguous and "+" is an invalid tag character.
// The first `err` returned will be displayed. No other helpText is displayed because it would be misleading.
//
// `helpText` is optional. If every result `ref` and `err` are nil, then helpText are combined in a final error.
// a parser may return constant helpText (e.g. "use '-' to read from input stream") or may return helpText
// that is related to a parse error on an ambiguous string (e.g. "use .git in the path if you meant git")
type result struct {
	// the first non-nil ref is returned. parser should set this if the location is parsed and valid.
	ref Reference
	// the first non-nil err is returned. parser should set this only when location unambiguously matches.
	err error
	// helpText is optional, and is shown if every parsers neither succeeded nor unambiguously failed
	helpText []string
}

// Option is a functional option for location parsing.
type Option func(*options)

// Options is a convenience to combine several options into one.
func Options(opts ...Option) Option {
	return func(opt *options) {
		for _, option := range opts {
			if option != nil {
				option(opt)
			}
		}
	}
}

// WithDefaultTag sets the default tag that will be used if one is not provided.
func WithContext(ctx context.Context) Option {
	return func(opts *options) {
		opts.ctx = ctx
	}
}

// WithStdin overrides the io.Reader used by the StdinParser
func WithStdin(reader io.Reader) Option {
	return func(opts *options) {
		opts.stdin = reader
	}
}

// WithStdin overrides the io.Writer used by the StdoutParser
func WithStdout(writer io.Writer) Option {
	return func(opts *options) {
		opts.stdout = writer
	}
}

func WithParsers(parsers ...parser) Option {
	return func(opts *options) {
		opts.parsers = append(opts.parsers, parsers...)
	}
}
