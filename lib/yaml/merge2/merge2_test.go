// Copyright 2019 Google LLC
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

package merge2_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/kio/filters"
	. "lib.kpt.dev/yaml/merge2"
)

var testCases = [][]testCase{scalarTestCases, listTestCases, elementTestCases, mapTestCases}

func TestMerge(t *testing.T) {
	for i := range testCases {
		for _, tc := range testCases[i] {
			actual, err := MergeStrings(tc.source, tc.dest)
			if !assert.NoError(t, err, tc.description) {
				t.FailNow()
			}
			e, err := filters.FormatInput(bytes.NewBufferString(tc.expected))
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			estr := strings.TrimSpace(e.String())
			a, err := filters.FormatInput(bytes.NewBufferString(actual))
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			astr := strings.TrimSpace(a.String())
			if !assert.Equal(t, estr, astr, tc.description) {
				t.FailNow()
			}
		}
	}
}

type testCase struct {
	description string
	source      string
	dest        string
	expected    string
}
