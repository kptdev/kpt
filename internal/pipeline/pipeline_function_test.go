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
package pipeline

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFunctionConfig(t *testing.T) {
	type input struct {
		name              string
		fn                v1alpha2.Function
		configFileContent string
		expected          string
	}

	cases := []input{
		{
			name:     "no config",
			fn:       v1alpha2.Function{},
			expected: "",
		},
		{
			name: "inline config",
			fn: v1alpha2.Function{
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
			fn:   v1alpha2.Function{},
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
			fn: v1alpha2.Function{
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
			cn, err := newFnConfig(&c.fn, os.TempDir())
			assert.NoError(t, err, "unexpected error")
			actual, err := cn.String()
			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, c.expected, actual, "unexpected result")
		})
	}
}

func TestValidateFunctionName(t *testing.T) {
	type input struct {
		Name  string
		Valid bool
	}
	inputs := []input{
		{
			"gcr.io/kpt-functions/generate-folders",
			true,
		},
		{
			"patch-strategic-merge",
			true,
		},
		{
			"a.b.c:1234/foo/bar/generate-folders",
			true,
		},
		{
			"ab-.b/c",
			false,
		},
		{
			"a/a/",
			false,
		},
		{
			"a//a/a",
			false,
		},
		{
			"example.com/.dots/myimage",
			false,
		},
		{
			"registry.io/foo/project--id.module--name.ver---sion--name",
			true,
		},
		{
			"Foo/FarB",
			false,
		},
	}

	for _, n := range inputs {
		n := n
		t.Run(n.Name, func(t *testing.T) {
			err := validateFunctionName(n.Name)
			if n.Valid && err != nil {
				t.Fatalf("function name %s should be valid", n.Name)
			}
			if !n.Valid && err == nil {
				t.Fatalf("function name %s should not be valid", n.Name)
			}
		})

	}
}

func TestValidatePath(t *testing.T) {
	type input struct {
		Path  string
		Valid bool
	}

	cases := []input{
		{
			"a/b/c",
			true,
		},
		{
			"a/b/",
			true,
		},
		{
			"/a/b",
			false,
		},
		{
			"./a",
			true,
		},
		{
			"./a/.../b",
			true,
		},
		{
			".",
			true,
		},
		{
			"a\\b",
			false,
		},
		{
			"a\b",
			false,
		},
		{
			"a\v",
			false,
		},
		{
			"a:\\b\\c",
			false,
		},
		{
			"../a/../b",
			true,
		},
		{
			"a//b",
			true,
		},
		{
			"a/b/.",
			true,
		},
		{
			"a/*/b",
			false,
		},
		{
			"./*",
			true,
		},
		{
			"a/b\\c",
			false,
		},
		{
			"././././",
			true,
		},
		{
			"./!&^%$/#(@)/_-=+|<;>?:'\"/'`",
			true,
		},
		{
			"",
			false,
		},
		{
			"\t \n",
			false,
		},
		{
			"*",
			false,
		},
	}

	for _, c := range cases {
		ret := validatePath(c.Path)
		if (ret == nil) != c.Valid {
			t.Fatalf("returned value for path %s should be %t, got %t",
				c.Path, c.Valid, (ret == nil))
		}
	}
}
