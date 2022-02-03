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
	"fmt"
	"strings"
)

func ParseReference(location string, opts ...Option) (Reference, error) {
	opt := makeOptions(opts...)

	var helpText []string
	for _, parser := range opt.parsers {
		result := parser(location, opt)
		if result.err != nil {
			return nil, result.err
		}
		if result.ref != nil {
			return result.ref, nil
		}
		helpText = append(helpText, result.helpText...)
	}

	return nil, fmt.Errorf("unable to parse location: %v", helpText)
}

func startsWith(value string, prefix string) (string, bool) {
	if parts := strings.SplitN(value, prefix, 2); len(parts) == 2 && len(parts[0]) == 0 {
		return parts[1], true
	}
	return prefix, false
}

// StdioParser enables "-" to resolve as io streams.
// Use location.WithStdin to provide the io.Reader.
// Use location.WithStdout to provide the io.Writer.
var StdioParser parser = func(value string, opt options) result {
	if value == "-" {
		switch {
		case opt.stdin != nil && opt.stdout != nil:
			return result{
				ref: DuplexStream{
					InputStream:  InputStream{Reader: opt.stdin},
					OutputStream: OutputStream{Writer: opt.stdout},
				},
			}
		case opt.stdin != nil:
			return result{
				ref: InputStream{Reader: opt.stdin},
			}
		case opt.stdout != nil:
			return result{
				ref: OutputStream{Writer: opt.stdout},
			}
		}
	}
	res := result{}
	if opt.stdin != nil {
		res.helpText = append(res.helpText, "Use '-' to read from input stream")
	}
	if opt.stdout != nil {
		res.helpText = append(res.helpText, "Use '-' to write to output stream")
	}
	return res
}

// GitParser enables standard parsing for the location.Git Reference type
var GitParser parser = func(value string, opt options) result {
	return result{
		ref:      parseGit(value, opt),
		helpText: []string{},
	}
}

// OciParser enables standard parsing for the location.Oci Reference type
var OciParser parser = func(value string, opt options) result {
	ref, err := parseOci(value)
	return result{
		ref: ref,
		err: err,
		helpText: []string{
			"OCI packages use 'oci://' prefix before standard image name",
		},
	}
}

// DirParser enables standard parsing for the location.Dir Reference type
var DirParser parser = func(value string, opt options) result {
	return result{
		ref: parseDir(value),
	}
}

// NewParser returns a parser for a custom Reference type.
// The returned parser is used in the location.WithParser Option
func NewParser(helpText []string, parser func(parse *Parse)) parser {
	return func(value string, opt options) result {
		req := Parse{
			Value: value,
			result: result{
				helpText: helpText,
			},
		}
		parser(&req)
		return req.result
	}
}

type Parse struct {
	result
	Value string
}

func (r *Parse) Result(ref Reference) {
	r.ref = ref
}

func (r *Parse) Fail(err error) {
	r.err = err
}

func (r *Parse) AddHelpText(helpText string) {
	r.helpText = append(r.helpText, helpText)
}
