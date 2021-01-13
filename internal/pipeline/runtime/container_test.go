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

func TestContainerRunner(t *testing.T) {
	// use sed to test instead of calling real docker
	var tests = []struct {
		input    string
		execPath string
		execArgs []string
		output   string
	}{
		{
			input: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`,
			execPath: "sed",
			execArgs: []string{"s/Deployment/StatefulSet/g"},
			output: `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: deployment-foo
---
apiVersion: v1
kind: Service
metadata:
  name: service-foo
`,
		},
	}

	for _, tt := range tests {
		instance := runtime.ContainerFn{}
		instance.Exec.Path = tt.execPath
		instance.Exec.Args = tt.execArgs
		input := bytes.NewBufferString(tt.input)
		output := &bytes.Buffer{}
		err := instance.Run(input, output)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		if !assert.Equal(t, tt.output, output.String()) {
			t.FailNow()
		}
	}
}
