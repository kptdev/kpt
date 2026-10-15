// Copyright 2026 The kpt Authors
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

package shlex

import (
	"fmt"
	"strings"
)

// Split splits an exec command string into individual arguments,
// respecting shell quoting and escape sequences.
// It handles:
//   - Single quotes (literal strings, no escapes)
//   - Double quotes (allows escapes for \, ", $, `, and newline)
//   - Unquoted arguments with space separation
//   - Escape sequences within quoted strings
func Split(s string) ([]string, error) {
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	wasQuoted := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		switch {
		case ch == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
			wasQuoted = true
		case ch == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
			wasQuoted = true
		case ch == '\\' && i+1 < len(s) && inDoubleQuote:
			// Handle escape sequences inside double quotes
			i++
			current.WriteByte(s[i])
		case (ch == ' ' || ch == '\t') && !inSingleQuote && !inDoubleQuote:
			// Whitespace outside quotes ends current argument
			if current.Len() > 0 || wasQuoted {
				args = append(args, current.String())
				current.Reset()
				wasQuoted = false
			}
		default:
			current.WriteByte(ch)
		}
	}

	if inSingleQuote {
		return nil, fmt.Errorf("unterminated single quote in command: %q", s)
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unterminated double quote in command: %q", s)
	}

	if current.Len() > 0 || wasQuoted {
		args = append(args, current.String())
	}

	return args, nil
}
