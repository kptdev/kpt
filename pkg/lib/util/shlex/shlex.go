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

// Split splits an exec command string into arguments.
// It follows the quoting, escaping, and comment rules of github.com/google/shlex:
// single quotes are literal, a backslash outside single quotes escapes the next
// character, and an unquoted # comments out the rest of the line.
// Unquoted spaces, tabs, carriage returns, and newlines separate arguments.
func Split(s string) ([]string, error) {
	args := make([]string, 0)
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	wasQuoted := false
	inComment := false

	for _, r := range s {
		switch {
		case inComment:
			if r == '\n' {
				inComment = false
			}
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\' && !inSingleQuote:
			escaped = true
		case r == '\'' && !inDoubleQuote:
			inSingleQuote = !inSingleQuote
			wasQuoted = true
		case r == '"' && !inSingleQuote:
			inDoubleQuote = !inDoubleQuote
			wasQuoted = true
		case r == '#' && !inSingleQuote && !inDoubleQuote && current.Len() == 0 && !wasQuoted:
			inComment = true
		case strings.ContainsRune(" \t\n\r", r) && !inSingleQuote && !inDoubleQuote:
			if current.Len() > 0 || wasQuoted {
				args = append(args, current.String())
				current.Reset()
				wasQuoted = false
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return args, fmt.Errorf("EOF found after escape character")
	}
	if inSingleQuote || inDoubleQuote {
		return args, fmt.Errorf("EOF found when expecting closing quote")
	}
	if !inComment && (current.Len() > 0 || wasQuoted) {
		args = append(args, current.String())
	}
	return args, nil
}
