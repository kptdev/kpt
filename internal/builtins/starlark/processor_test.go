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

package starlark

import (
	"testing"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
)

func TestProcess_SetNamespace(t *testing.T) {
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
  kind: StarlarkRun
  metadata:
    name: set-namespace
  source: |
    def run(r, ns_value):
      for resource in r:
        resource["metadata"]["namespace"] = ns_value
    run(ctx.resource_list["items"], "new-namespace")
`
	rl, err := fn.ParseResourceList([]byte(input))
	assert.NoError(t, err)

	ok, err := Process(rl)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "new-namespace", rl.Items[0].GetNamespace())
}

func TestProcess_InvalidScript(t *testing.T) {
	input := `apiVersion: config.kubernetes.io/v1
kind: ResourceList
items: []
functionConfig:
  apiVersion: fn.kpt.dev/v1alpha1
  kind: StarlarkRun
  metadata:
    name: bad-script
  source: |
    this is not valid starlark!!!
`
	rl, err := fn.ParseResourceList([]byte(input))
	assert.NoError(t, err)

	ok, err := Process(rl)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.Len(t, rl.Results, 1)
}
