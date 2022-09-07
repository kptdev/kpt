// Copyright 2022 Google LLC
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

package copy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextRevisionNumber(t *testing.T) {
	testcases := map[string]struct {
		input    string
		expected string
		err      string
	}{
		"invalid": {
			input: "alskdfj",
			err:   "invalid revision format alskdfj; explicit --revision flag is required",
		},
		"no dots": {
			input:    "v4",
			expected: "v5",
		},
		"one dot": {
			input:    "v3.1",
			expected: "v3.2",
		},
		"two dots": {
			input:    "v1.2.3",
			expected: "v1.2.4",
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			output, err := nextRevisionNumber(tc.input)
			if tc.err != "" {
				require.Error(t, err)
				require.Equal(t, tc.err, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, output)
			}
		})
	}
}
