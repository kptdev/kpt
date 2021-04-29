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

package fnerrors

import (
	"fmt"
	"strings"
)

const (
	FnExecErrorTruncateLines = 4
	// FnExecErrorIndentation is the number of spaces at the beginning of each
	// line of function failure messages.
	FnExecErrorIndentation = 2
)

// FnExecError contains the information about the function failure that will
// be outputted.
type FnExecError struct {
	// DisableOutputTruncate indicates should error message truncation be disabled
	DisableOutputTruncate bool
	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`
	// TODO: add Results after structured results are supported
}

// String returns string representation of the failure.
func (fe *FnExecError) String() string {
	var b strings.Builder
	b.WriteString("Stderr:\n")

	lineIndent := strings.Repeat(" ", 2)
	if fe.DisableOutputTruncate {
		// stderr string should have indentations
		for _, s := range strings.Split(fe.Stderr, "\n") {
			b.WriteString(fmt.Sprintf(lineIndent+"%q\n", s))
		}
	} else {
		printedLines := 0
		lines := strings.Split(fe.Stderr, "\n")
		for i, s := range lines {
			if i >= FnExecErrorTruncateLines {
				break
			}
			b.WriteString(fmt.Sprintf(lineIndent+"%q\n", s))
			printedLines++
		}
		if printedLines < len(lines) {
			b.WriteString(fmt.Sprintf(lineIndent+"...(%d line(s) truncated)\n", len(lines)-printedLines))
		}
	}
	b.WriteString(fmt.Sprintf("Exit Code: %d\n", fe.ExitCode))
	return b.String()
}

func (fe *FnExecError) Error() string {
	return fe.String()
}
