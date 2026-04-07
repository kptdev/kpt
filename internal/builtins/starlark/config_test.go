// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package starlark

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestStarlarkConfig(t *testing.T) {
	testcases := []struct {
		name         string
		config       string
		expectErrMsg string
	}{
		{
			name: "valid Run",
			config: `apiVersion: fn.kpt.dev/v1alpha1
kind: Run
metadata:
  name: my-star-fn
  namespace: foo
source: |
  def run(r, ns_value):
    for resource in r:
      resource["metadata"]["namespace"] = ns_value
  run(ctx.resource_list["items"], "baz")
`,
		},
		{
			name: "Run missing Source",
			config: `apiVersion: fn.kpt.dev/v1alpha1
kind: Run
metadata:
  name: my-star-fn
`,
			expectErrMsg: "`source` must not be empty",
		},
		{
			name: "valid ConfigMap",
			config: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-star-fn
data:
  source: |
    def run(r, ns_value):
      for resource in r:
        resource["metadata"]["namespace"] = ns_value
    run(ctx.resource_list["items"], "baz")
`,
		},
		{
			name: "ConfigMap missing source",
			config: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-star-fn
`,
			expectErrMsg: "`source` must not be empty",
		},
		{
			name: "ConfigMap with parameter but missing source",
			config: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-star-fn
data:
  param1: foo
`,
			expectErrMsg: "`source` must not be empty",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sr := &Run{}
			node, err := yaml.Parse(tc.config)
			assert.NoError(t, err)

			err = sr.Config(node)
			if tc.expectErrMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrMsg)
			}
		})
	}
}
