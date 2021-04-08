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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package cmdrender

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFunctionConfig(t *testing.T) {
	type input struct {
		name              string
		fn                kptfilev1alpha2.Function
		configFileContent string
		expected          string
	}

	cases := []input{
		{
			name:     "no config",
			fn:       kptfilev1alpha2.Function{},
			expected: "",
		},
		{
			name: "inline config",
			fn: kptfilev1alpha2.Function{
				Config: *yaml.MustParse(`apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy`).YNode(),
			},
			expected: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy
`,
		},
		{
			name: "file config",
			fn:   kptfilev1alpha2.Function{},
			configFileContent: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy`,
			expected: `apiVersion: cft.dev/v1alpha1
kind: ResourceHierarchy
metadata:
  name: root-hierarchy
  namespace: hierarchy
`,
		},
		{
			name: "map config",
			fn: kptfilev1alpha2.Function{
				ConfigMap: map[string]string{
					"foo": "bar",
				},
			},
			expected: `apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {foo: bar}
`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if c.configFileContent != "" {
				tmp, err := ioutil.TempFile("", "kpt-pipeline-*")
				assert.NoError(t, err, "unexpected error")
				_, err = tmp.WriteString(c.configFileContent)
				assert.NoError(t, err, "unexpected error")
				c.fn.ConfigPath = path.Base(tmp.Name())
			}
			cn, err := newFnConfig(&c.fn, types.UniquePath(os.TempDir()))
			assert.NoError(t, err, "unexpected error")
			actual, err := cn.String()
			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, c.expected, actual, "unexpected result")
		})
	}
}
