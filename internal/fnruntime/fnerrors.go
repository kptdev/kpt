// Copyright 2021 The kpt Authors
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

package fnruntime

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

// ExecError implements an error type that stores information about function failure.
type ExecError struct {
	// OriginalErr is the original error returned from function runtime
	OriginalErr error

	// TruncateOutput indicates should error message be truncated
	TruncateOutput bool

	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`

	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`
}

// String returns string representation of the failure.
func (fe *ExecError) String() string {
	var b strings.Builder

	errLines := &MultiLineFormatter{
		Title:          "Stderr",
		Lines:          strings.Split(fe.Stderr, "\n"),
		UseQuote:       true,
		TruncateOutput: fe.TruncateOutput,
	}
	b.WriteString(errLines.String())
	b.WriteString(fmt.Sprintf("  Exit Code: %d\n", fe.ExitCode))
	return b.String()
}

func (fe *ExecError) Error() string {
	return fe.String()
}
