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
