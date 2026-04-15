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

package fnruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinesToSlice(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected []string
	}{
		"empty": {
			input:    "",
			expected: nil,
		},
		"whitespace only": {
			input:    "  \t\n  ",
			expected: nil,
		},
		"single line": {
			input:    "v1.0.0",
			expected: []string{"v1.0.0"},
		},
		"single line with surrounding space": {
			input:    "  v1.0.0  \n",
			expected: []string{"v1.0.0"},
		},
		"multiple lines": {
			input:    "v1.0.0\nv1.0.1\nv1.0.2",
			expected: []string{"v1.0.0", "v1.0.1", "v1.0.2"},
		},
		"windows line endings": {
			input:    "a\r\nb\r\nc",
			expected: []string{"a", "b", "c"},
		},
		"blank line not preserved": {
			input:    "a\n\nb",
			expected: []string{"a", "b"},
		},
		"per-line trim": {
			input:    "  foo  \n  bar  ",
			expected: []string{"foo", "bar"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := linesToSlice(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestRegClientLister_Name(t *testing.T) {
	l := &RegClientLister{}
	assert.Equal(t, "regclient", l.Name())
}

func TestLocalLister_Name(t *testing.T) {
	tests := map[string]struct {
		binary   string
		expected string
	}{
		"docker": {
			binary:   "docker",
			expected: "local-docker",
		},
		"empty binary": {
			binary:   "",
			expected: "local-",
		},
		"path-like binary": {
			binary:   "/usr/local/bin/nerdctl",
			expected: "local-/usr/local/bin/nerdctl",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := &LocalLister{Binary: tc.binary}
			assert.Equal(t, tc.expected, l.Name())
		})
	}
}
