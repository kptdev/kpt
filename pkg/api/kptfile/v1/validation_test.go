// Copyright 2021 The kpt Authors
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
package v1

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestKptfileValidate(t *testing.T) {
	type input struct {
		name    string
		kptfile KptFile
		valid   bool
	}

	cases := []input{
		{
			name: "pipeline: empty",
			kptfile: KptFile{
				Pipeline: &Pipeline{},
			},
			valid: true,
		},
		{
			name: "pipeline: validcase",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image: "patch-strategic-merge",
						},
						{
							Image: "gcr.io/kpt-fn/set-annotations:v0.1",
							ConfigMap: map[string]string{
								"environment": "dev",
							},
						},
					},
					Validators: []Function{
						{
							Image: "gcr.io/kpt-fn/gatekeeper",
						},
					},
				},
			},
			valid: true,
		},
		{
			name: "pipeline: invalid image name",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image: "patch@_@strategic-merge",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "pipeline: more than 1 config",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image:      "image",
							ConfigPath: "./config.yaml",
							ConfigMap: map[string]string{
								"foo": "bar",
							},
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "pipeline: absolute config path",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image:      "image",
							ConfigPath: "/config.yaml",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "pipeline: configpath referring file in parent",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image:      "image",
							ConfigPath: "../config.yaml",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "pipeline: cleaned configpath contains ..",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image:      "image",
							ConfigPath: "a/b/../../../config.yaml",
						},
					},
				},
			},
			valid: false,
		},
		{
			name: "pipeline: configpath contains invalid .. references",
			kptfile: KptFile{
				Pipeline: &Pipeline{
					Mutators: []Function{
						{
							Image:      "image",
							ConfigPath: "a/.../config.yaml",
						},
					},
				},
			},
			valid: false,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := c.kptfile.Validate(filesys.FileSystemOrOnDisk{}, "")
			if c.valid && err != nil {
				t.Fatalf("kptfile should be valid, %s", err)
			}
			if !c.valid && err == nil {
				t.Fatal("kptfile should not be valid")
			}
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
			"gcr.io/kpt-fn/generate-folders",
			true,
		},
		{
			"patch-strategic-merge",
			true,
		},
		{
			"gcr.io/kpt-fn/generate-folders:unstable",
			true,
		},
		{
			"patch-strategic-merge:v1.3_beta",
			true,
		},
		{
			"gcr.io/kpt-fn/generate-folders:v1.2.3-alpha1",
			true,
		},
		{
			"patch-strategic-merge:x.y.z",
			true,
		},
		{
			"patch-strategic-merge::@!",
			false,
		},
		{
			"patch-strategic-merge:",
			false,
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
		{
			"example.com/foo/generate-folders@sha256:3434a5299f8fcb2c2ade9975e56ca5a622427b9d5a9a971640765e830fb90a0e",
			true,
		},
	}

	for _, n := range inputs {
		n := n
		t.Run(n.Name, func(t *testing.T) {
			err := ValidateFunctionImageURL(n.Name)
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
			false,
		},
		{
			".",
			true,
		},
		{
			"a\\b",
			true,
		},
		{
			"a\b",
			true,
		},
		{
			"a\v",
			true,
		},
		{
			"a:\\b\\c",
			true,
		},
		{
			"../a/../b",
			false,
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
			"././././",
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
			"a/b/../config.yaml",
			true,
		},
	}

	for _, c := range cases {
		ret := validateFnConfigPathSyntax(c.Path)
		if (ret == nil) != c.Valid {
			t.Fatalf("returned value for path %q should be %t, got %t",
				c.Path, c.Valid, (ret == nil))
		}
	}
}

func TestIsKustomization(t *testing.T) {
	testcases := []struct {
		name  string
		input string
		exp   bool
	}{
		{
			"resource in a kustomization file is a kustomization",
			`
metadata:
  annotations:
    config.kubernetes.io/path: kustomization.yaml
`,
			true,
		},
		{
			"resource in a kustomization file with .yml extn is a kustomization",
			`
metadata:
  annotations:
    config.kubernetes.io/path: kustomization.yml
`,
			true,
		},
		{
			"resource in a kustomization file in a subdir is a kustomization",
			`
metadata:
  annotations:
    config.kubernetes.io/path: subdir/kustomization.yaml
`,
			true,
		},
		{
			"resource in a non-kustomization file, with empty apigroup and Kustomization kind is a kustomization",
			`apiVersion:
kind: Kustomization
`,
			true,
		},
		{
			"resource in a non-kustomization file with Kustomization APIGroup is a kustomization",
			`apiVersion: kustomize.config.k8s.io/v1beta1
`,
			true,
		},
		{
			"resource in a non-kustomization file with non-kustomize apigroup is not a kustomization",
			`apiVersion: non.kubernetes.io/v1
`,
			false,
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := isKustomization(yaml.MustParse(tc.input))
			if got != tc.exp {
				t.Fatalf("got %v expected %v", got, tc.exp)
			}
		})
	}
}

func TestGetValidatedFnConfigFromPath(t *testing.T) {
	testcases := []struct {
		name   string
		input  string
		exp    string
		errMsg string
	}{
		{
			name: "normal resource",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: myService
spec:
  selector:
    app: bar
`,
			exp: `apiVersion: v1
kind: Service
metadata:
  name: myService
  annotations:
    config.kubernetes.io/index: '0'
    internal.config.kubernetes.io/index: '0'
    internal.config.kubernetes.io/seqindent: 'compact'
spec:
  selector:
    app: bar
`,
		},
		{
			name: "multiple resources wrapped in List",
			input: `
apiVersion: v1
kind: List
metadata:
  name: upsert-multiple-resources-config
items:
- apiVersion: v1
  kind: Service
  metadata:
    name: myService
    namespace: mySpace
  spec:
    selector:
      app: bar
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: myDeployment2
    namespace: mySpace
  spec:
    replicas: 10
`,
			exp: `apiVersion: v1
kind: List
metadata:
  name: upsert-multiple-resources-config
  annotations:
    config.kubernetes.io/index: '0'
    internal.config.kubernetes.io/index: '0'
    internal.config.kubernetes.io/seqindent: 'compact'
items:
- apiVersion: v1
  kind: Service
  metadata:
    name: myService
    namespace: mySpace
  spec:
    selector:
      app: bar
- apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: myDeployment2
    namespace: mySpace
  spec:
    replicas: 10
`,
		},
		{
			name: "error for multiple resources",
			input: `
apiVersion: v1
kind: Service
metadata:
  name: myService
  namespace: mySpace
---
apiVersion: v1
kind: Service
metadata:
  name: myService2
  namespace: mySpace
`,
			errMsg: `functionConfig "f1.yaml" must not contain more than one config, got 2`,
		},
	}
	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			err := os.WriteFile(filepath.Join(d, "f1.yaml"), []byte(tc.input), 0700)
			assert.NoError(t, err)
			got, err := GetValidatedFnConfigFromPath(filesys.FileSystemOrOnDisk{}, types.UniquePath(d), "f1.yaml")
			if tc.errMsg != "" {
				assert.Error(t, err)
				assert.Equal(t, tc.errMsg, err.Error())
				return
			}
			assert.NoError(t, err)
			actual, err := got.String()
			assert.NoError(t, err)
			assert.Equal(t, tc.exp, actual)
		})
	}
}
