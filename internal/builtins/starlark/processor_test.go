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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func parseResourceList(t *testing.T, input string) *framework.ResourceList {
	t.Helper()
	rw := &kio.ByteReader{Reader: strings.NewReader(input)}
	nodes, err := rw.Read()
	assert.NoError(t, err)

	rl := &framework.ResourceList{Items: nodes}
	node, err := yaml.Parse(input)
	if err == nil {
		fc := node.Field("functionConfig")
		if fc != nil && fc.Value != nil {
			rl.FunctionConfig = fc.Value
		}
	}
	return rl
}

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
	rw := &kio.ByteReadWriter{
		Reader:             strings.NewReader(input),
		WrappingAPIVersion: kio.ResourceListAPIVersion,
		WrappingKind:       kio.ResourceListKind,
	}
	nodes, err := rw.Read()
	assert.NoError(t, err)

	rl := &framework.ResourceList{
		Items:          nodes,
		FunctionConfig: rw.FunctionConfig,
	}

	err = Process(rl)
	assert.NoError(t, err)

	ns, err := rl.Items[0].GetString("metadata.namespace")
	assert.NoError(t, err)
	assert.Equal(t, "new-namespace", ns)
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
	rw := &kio.ByteReadWriter{
		Reader:             strings.NewReader(input),
		WrappingAPIVersion: kio.ResourceListAPIVersion,
		WrappingKind:       kio.ResourceListKind,
	}
	nodes, err := rw.Read()
	assert.NoError(t, err)

	rl := &framework.ResourceList{
		Items:          nodes,
		FunctionConfig: rw.FunctionConfig,
	}

	err = Process(rl)
	assert.Error(t, err)
	assert.Len(t, rl.Results, 1)
}
