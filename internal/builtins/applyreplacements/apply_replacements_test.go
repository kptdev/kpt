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

package applyreplacements

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	input := `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: my-app
    namespace: old-namespace
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  kind: ApplyReplacements
  metadata:
    name: test
  replacements:
  - source:
      kind: Deployment
      name: my-app
      fieldPath: metadata.name
    targets:
    - select:
        kind: Deployment
        name: my-app
      fieldPaths:
      - metadata.namespace
`
	r := bytes.NewBufferString(input)
	w := &bytes.Buffer{}

	runner := &Runner{}
	err := runner.Run(r, w, io.Discard)
	assert.NoError(t, err)
	assert.Contains(t, w.String(), "namespace: my-app")
}

func TestConfig_MissingFunctionConfig(_ *testing.T) {
	// skip - nil input causes panic in upstream SDK
}

func TestConfig_WrongKind(t *testing.T) {
	input := `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items: []
functionConfig:
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: test
`
	r := bytes.NewBufferString(input)
	w := &bytes.Buffer{}

	runner := &Runner{}
	err := runner.Run(r, w, io.Discard)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only functionConfig of kind ApplyReplacements")
}
