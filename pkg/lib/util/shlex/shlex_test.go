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
			expected: []string{},
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: []string{},
		},
		{
			name:     "newlines and carriage returns only",
			input:    "\n\r\t",
			expected: []string{},
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
			expected:  []string{},
			wantError: true,
			errorMsg:  "EOF found when expecting closing quote",
		},
		{
			name:      "unterminated single quote",
			input:     `'foo bar`,
			expected:  []string{},
			wantError: true,
			errorMsg:  "EOF found when expecting closing quote",
		},
		{
			name:      "words before an unterminated quote are returned",
			input:     `ok "bad`,
			expected:  []string{"ok"},
			wantError: true,
			errorMsg:  "EOF found when expecting closing quote",
		},
		{
			name:      "trailing backslash",
			input:     `foo\`,
			expected:  []string{},
			wantError: true,
			errorMsg:  "EOF found after escape character",
		},
		{
			name:      "words before a trailing backslash are returned",
			input:     `pre foo\`,
			expected:  []string{"pre"},
			wantError: true,
			errorMsg:  "EOF found after escape character",
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
			name:     "escaped space joins unquoted arguments",
			input:    `foo\ bar baz`,
			expected: []string{"foo bar", "baz"},
		},
		{
			name:     "escaped quote in an unquoted argument",
			input:    `foo\"bar`,
			expected: []string{`foo"bar`},
		},
		{
			name:     "hash inside a word is literal",
			input:    "foo#bar",
			expected: []string{"foo#bar"},
		},
		{
			name:     "comment is omitted",
			input:    "foo # comment",
			expected: []string{"foo"},
		},
		{
			name:     "comment ends at newline",
			input:    "foo # comment\nbar",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "newline and carriage return separate arguments",
			input:    "foo\nbar\rbaz",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "newline inside double quotes is preserved",
			input:    "\"foo\nbar\"",
			expected: []string{"foo\nbar"},
		},
		{
			name:     "backslash escapes the next character inside double quotes",
			input:    `"foo\nbar"`,
			expected: []string{"foonbar"},
		},
		{
			name:  "google shlex example",
			input: "one two \"three four\" \"five \\\"six\\\"\" seven#eight # nine # ten\n eleven 'twelve\\' thirteen=13 fourteen/14",
			expected: []string{
				"one", "two", "three four", "five \"six\"", "seven#eight",
				"eleven", "twelve\\", "thirteen=13", "fourteen/14",
			},
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
			name:      "escaped closing quote leaves the double quote unterminated",
			input:     `"foo\"`,
			expected:  []string{},
			wantError: true,
			errorMsg:  "EOF found when expecting closing quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Split(tt.input)

			assert.Equal(t, tt.expected, result)
			if tt.wantError {
				assert.ErrorContains(t, err, tt.errorMsg)
				return
			}
			assert.NoError(t, err)
		})
	}
}
