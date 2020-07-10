// Copyright 2020 Google LLC
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

package orchestrators

import (
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/GoogleContainerTools/kpt/internal/cmdexport/types"
)

type tektonTaskTestCase struct {
	description string
	config      *TektonTaskConfig
	expected    string
}

var tektonTaskTestCases = []tektonTaskTestCase{
	{
		description: "generate a tekton task",
		config: &TektonTaskConfig{
			PipelineConfig: &types.PipelineConfig{
				Dir: "local-resources/",
			},
			Name: "run-kpt-functions",
		},
		expected: `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: source
        mountPath: /source
    steps:
      - name: run-kpt-functions
        image: gongpu/kpt:latest
        args:
          - fn
          - run
          - $(workspaces.source.path)/local-resources
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
`,
	},
	{
		description: "generate a tekton task with --fn-path",
		config: &TektonTaskConfig{
			PipelineConfig: &types.PipelineConfig{
				Dir:     "local-resources",
				FnPaths: []string{"config/"},
			},
			Name: "run-kpt-functions",
		},
		expected: `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: source
        mountPath: /source
    steps:
      - name: run-kpt-functions
        image: gongpu/kpt:latest
        args:
          - fn
          - run
          - $(workspaces.source.path)/local-resources
          - --fn-path
          - $(workspaces.source.path)/config
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
`,
	},
	{
		description: "generate a tekton task with multiple --fn-path",
		config: &TektonTaskConfig{
			PipelineConfig: &types.PipelineConfig{
				Dir:     "local-resources",
				FnPaths: []string{"config/gate-keeper.yaml", "config/label-namespace.yaml"},
			},
			Name: "run-kpt-functions",
		},
		expected: `
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: source
        mountPath: /source
    steps:
      - name: run-kpt-functions
        image: gongpu/kpt:latest
        args:
          - fn
          - run
          - $(workspaces.source.path)/local-resources
          - --fn-path
          - $(workspaces.source.path)/config/gate-keeper.yaml
          - --fn-path
          - $(workspaces.source.path)/config/label-namespace.yaml
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
`,
	},
}

func TestTektonTask(t *testing.T) {
	for i := range tektonTaskTestCases {
		testCase := tektonTaskTestCases[i]

		t.Run(testCase.description, func(t *testing.T) {
			pipeline, _ := new(TektonTask).Init(testCase.config).Generate()

			actual := string(pipeline)
			expected := strings.TrimLeft(testCase.expected, "\n")

			assert.Equal(t, expected, actual)
		})
	}
}
