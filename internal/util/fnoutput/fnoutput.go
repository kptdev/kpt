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

package fnoutput

import (
	"fmt"
	"strings"
)

// FnFailure contains the information about the function failure that will
// be outputted.
type FnFailure struct {
	// Stderr is the content written to function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code returned from function
	ExitCode int `yaml:"exitCode,omitempty"`
	// TODO: add Results after structured results are supported
}

const truncateLines = 4

// String returns string representation of the failure. truncate is used
// to control whether the contents will be truncated.
func (ff *FnFailure) String(truncate bool) (string, error) {
	var b strings.Builder
	_, err := b.WriteString("Stderr:\n")
	if err != nil {
		return "", fmt.Errorf("failed to get function failure output: %w", err)
	}

	if !truncate {
		// stderr string should have indentations
		for _, s := range strings.Split(ff.Stderr, "\n") {
			_, err := b.WriteString("  " + s + "\n")
			if err != nil {
				return "", fmt.Errorf("failed to get function failure output: %w", err)
			}
		}
	} else {
		printedLines := 0
		lines := strings.Split(ff.Stderr, "\n")
		for i, s := range lines {
			if i >= truncateLines {
				break
			}
			_, err := b.WriteString("  " + s + "\n")
			if err != nil {
				return "", fmt.Errorf("failed to get function failure output: %w", err)
			}
			printedLines++
		}
		if printedLines < len(lines) {
			_, err := b.WriteString(fmt.Sprintf("  ...(%d line(s) truncated)\n", len(lines)-printedLines))
			if err != nil {
				return "", fmt.Errorf("failed to get function failure output: %w", err)
			}
		}
	}
	_, err = b.WriteString(fmt.Sprintf("Exit Code: %d\n", ff.ExitCode))
	if err != nil {
		return "", fmt.Errorf("failed to get function failure output: %w", err)
	}
	return b.String(), nil
}
