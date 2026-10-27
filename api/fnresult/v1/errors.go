// Copyright 2021,2026 The kpt Authors
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

package v1

import (
	"fmt"
	"strings"
	"time"

	apiutils "github.com/kptdev/kpt/api/utils"
)

const (
	FnExecErrorTruncateLines = 4
	// FnExecErrorIndentation is the number of spaces at the beginning of each
	// line of function failure messages.
	FnExecErrorIndentation = 2
)

// ExecError implements an error type that stores information about function failure.
// +kubebuilder:object:generate=false
// +kubebuilder:ac:generate=false
type ExecError struct {
	// OriginalErr is the original error returned from function runtime
	OriginalErr error

	// TruncateOutput indicates should error message be truncated
	TruncateOutput bool

	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`

	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`

	// Timeout is the time limit that was exceeded. It is zero if the
	// function did not time out.
	Timeout time.Duration `yaml:"-"`

	// RunTimeout is true if Timeout is the time limit of the whole run, and
	// false if it is the time limit of this function.
	RunTimeout bool `yaml:"-"`
}

// String returns string representation of the failure.
func (fe *ExecError) String() string {
	var b strings.Builder

	errLine := &apiutils.SingleLineFormatter{
		Title:     "[Stderr]",
		Lines:     strings.Split(fe.Stderr, "\n"),
		UseQuote:  false,
		Separator: ", ",
	}
	b.WriteString(errLine.String())
	fmt.Fprintf(&b, "  Exit Code: %d\n", fe.ExitCode)
	if msg := fe.TimeoutMessage(); msg != "" {
		fmt.Fprintf(&b, "  Timed Out: %s\n", msg)
	}
	return b.String()
}

// TimeoutMessage describes the time limit that was exceeded. It is empty if
// the function did not time out.
func (fe *ExecError) TimeoutMessage() string {
	switch {
	case fe.Timeout == 0:
		return ""
	case fe.RunTimeout:
		return fmt.Sprintf("pipeline execution timeout of %v exceeded", fe.Timeout)
	default:
		return fmt.Sprintf("function execution timeout of %v exceeded", fe.Timeout)
	}
}

func (fe *ExecError) Error() string {
	return fe.String()
}
