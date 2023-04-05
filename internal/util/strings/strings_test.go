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

package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinStringsWithQuotes(t *testing.T) {
	testCases := map[string]struct {
		slice    []string
		expected string
	}{
		"empty slice": {
			slice:    []string{},
			expected: ``,
		},
		"single element": {
			slice:    []string{"a"},
			expected: `"a"`,
		},
		"two elements": {
			slice:    []string{"a", "b"},
			expected: `"a", "b"`,
		},
		"three elements": {
			slice:    []string{"a", "b", "c"},
			expected: `"a", "b", "c"`,
		},
		"multiple elements": {
			slice:    []string{"a", "b", "c", "d", "e"},
			expected: `"a", "b", "c", "d", "e"`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			res := JoinStringsWithQuotes(tc.slice)
			assert.Equal(t, tc.expected, res)
		})
	}
}
