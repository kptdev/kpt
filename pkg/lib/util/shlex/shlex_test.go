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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitExecArgs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		wantError bool
		errorMsg  string
	}{
		{
			name:     "simple arguments",
			input:    "foo bar baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "single argument",
			input:    "foo",
			expected: []string{"foo"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "tabs and spaces",
			input:    "foo\t\tbar   baz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "double quoted argument with spaces",
			input:    `"foo bar"`,
			expected: []string{"foo bar"},
		},
		{
			name:     "single quoted argument with spaces",
			input:    `'foo bar'`,
			expected: []string{"foo bar"},
		},
		{
			name:     "mixed quotes and unquoted",
			input:    `/bin/sh -c 'echo hello'`,
			expected: []string{"/bin/sh", "-c", "echo hello"},
		},
		{
			name:     "double quoted with escaped quote",
			input:    `"foo\"bar"`,
			expected: []string{`foo"bar`},
		},
		{
			name:     "double quoted with escaped backslash",
			input:    `"foo\\bar"`,
			expected: []string{`foo\bar`},
		},
		{
			name:     "single quoted preserves backslash literally",
			input:    `'foo\nbar'`,
			expected: []string{`foo\nbar`},
		},
		{
			name:     "multiple quoted arguments",
			input:    `"first arg" 'second arg' third`,
			expected: []string{"first arg", "second arg", "third"},
		},
		{
			name:     "quoted argument adjacent to unquoted",
			input:    `foo"bar"baz`,
			expected: []string{"foobarbaz"},
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  foo bar  ",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "path with spaces",
			input:    `"/usr/local/bin/my program"`,
			expected: []string{"/usr/local/bin/my program"},
		},
		{
			name:     "empty double quoted string",
			input:    `""`,
			expected: []string{""},
		},
		{
			name:     "empty single quoted string",
			input:    `''`,
			expected: []string{""},
		},
		{
			name:     "escaped space in double quotes",
			input:    `"foo\ bar"`,
			expected: []string{"foo bar"},
		},
		{
			name:      "unterminated double quote",
			input:     `"foo bar`,
			wantError: true,
			errorMsg:  "unterminated double quote",
		},
		{
			name:      "unterminated single quote",
			input:     `'foo bar`,
			wantError: true,
			errorMsg:  "unterminated single quote",
		},
		{
			name:     "mixed nested quotes (single in double)",
			input:    `"it's working"`,
			expected: []string{"it's working"},
		},
		{
			name:     "mixed nested quotes (double in single)",
			input:    `'he said "hello"'`,
			expected: []string{`he said "hello"`},
		},
		{
			name:     "backslash only in unquoted context",
			input:    `foo\ bar baz`,
			expected: []string{`foo\`, "bar", "baz"},
		},
		{
			name:     "complex real-world command",
			input:    `/bin/bash -c 'echo "Hello World"'`,
			expected: []string{"/bin/bash", "-c", `echo "Hello World"`},
		},
		{
			name:     "multiple escaped characters",
			input:    `"a\"b\\c"`,
			expected: []string{`a"b\c`},
		},
		{
			name:      "single trailing backslash in double quote",
			input:     `"foo\"`,
			wantError: true,
			errorMsg:  "unterminated double quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Split(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
