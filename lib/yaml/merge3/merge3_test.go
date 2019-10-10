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

package merge3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	. "lib.kpt.dev/yaml/merge3"
)

var testCases = [][]testCase{scalarTestCases, listTestCases, mapTestCases, elementTestCases}

func TestMerge(t *testing.T) {
	for i := range testCases {
		for _, tc := range testCases[i] {
			actual, err := MergeStrings(tc.local, tc.origin, tc.update)
			if tc.err == nil {
				if !assert.NoError(t, err, tc.description) {
					t.FailNow()
				}
				if !assert.Equal(t,
					strings.TrimSpace(tc.expected), strings.TrimSpace(actual), tc.description) {
					t.FailNow()
				}
			} else {
				if !assert.Errorf(t, err, tc.description) {
					t.FailNow()
				}
				if !assert.Contains(t, tc.err.Error(), err.Error()) {
					t.FailNow()
				}
			}
		}
	}
}

type testCase struct {
	description string
	origin      string
	update      string
	local       string
	expected    string
	err         error
}
