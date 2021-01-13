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

package runtime_test

import (
	"bytes"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pipeline/runtime"
	"github.com/stretchr/testify/assert"
)

func TestExecRunner(t *testing.T) {
	var tests = []struct {
		name           string
		input          string
		expectedOutput string
		expectedError  string
		instance       runtime.ExecFn
	}{
		{
			name: "exec_sed",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`,
			expectedOutput: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`,
			expectedError: "",
			instance: runtime.ExecFn{
				Path: "sed",
				Args: []string{"s/Deployment/StatefulSet/g"},
			},
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			input := bytes.NewBufferString(tt.input)
			output := &bytes.Buffer{}

			// run the function
			err := tt.instance.Run(input, output)

			// check for errors
			if tt.expectedError != "" {
				if !assert.EqualError(t, err, tt.expectedError) {
					t.FailNow()
				}
				return
			}
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// verify the output
			actual := output.String()
			if !assert.Equal(t, tt.expectedOutput, actual) {
				t.FailNow()
			}
		})
	}
}
