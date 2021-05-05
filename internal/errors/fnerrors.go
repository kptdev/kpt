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

package errors

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
	// OriginalErr is the original error returned from function runtime
	OriginalErr error

	// TruncateOutput indicates should error message be truncated
	TruncateOutput bool

	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`

	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`

	// TODO: This introduces import cycle between errors and fnresult package.
	// Will require moving fnErrors outside errors package.
	// FnResult is the structured result returned from the function
	// FnResult *fnresult.Result
}

// String returns string representation of the failure.
func (fe *FnExecError) String() string {
	var b strings.Builder
	universalIndent := strings.Repeat(" ", FnExecErrorIndentation)
	b.WriteString(universalIndent + "Stderr:\n")

	lineIndent := strings.Repeat(" ", FnExecErrorIndentation+2)
	if !fe.TruncateOutput {
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
	b.WriteString(fmt.Sprintf(universalIndent+"Exit Code: %d\n", fe.ExitCode))
	return b.String()
}

func (fe *FnExecError) Error() string {
	return fe.String()
}
