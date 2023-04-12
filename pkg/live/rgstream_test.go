// Copyright 2020 The kpt Authors
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

package live

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestResourceStreamManifestReader_Read(t *testing.T) {
	_ = apiextv1.AddToScheme(scheme.Scheme)
	testCases := map[string]struct {
		manifests      map[string]string
		namespace      string
		expectedObjs   object.ObjMetadataSet
		expectedErrMsg string
	}{
		"Kptfile is excluded": {
			manifests: map[string]string{
				"Kptfile": kptFile,
			},
			namespace:    "test-namespace",
			expectedObjs: []object.ObjMetadata{},
		},
		"Only a pod is valid": {
			manifests: map[string]string{
				"pod-a.yaml": podA,
			},
			namespace: "test-namespace",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "Pod",
					},
					Name:      "pod-a",
					Namespace: "test-namespace",
				},
			},
		},
		"Multiple resources are valid": {
			manifests: map[string]string{
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			namespace: "test-namespace",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Kind: "Pod",
					},
					Name:      "pod-a",
					Namespace: "test-namespace",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "test-namespace",
				},
			},
		},
		"CR and CRD in the same set is ok": {
			manifests: map[string]string{
				"crd.yaml": crd,
				"cr.yaml":  cr,
			},
			namespace: "test-namespace",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "custom.io",
						Kind:  "Custom",
					},
					Name: "cr",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apiextensions.k8s.io",
						Kind:  "CustomResourceDefinition",
					},
					Name: "custom.io",
				},
			},
		},
		"CR with unknown type is not allowed": {
			manifests: map[string]string{
				"cr.yaml": cr,
			},
			namespace:      "test-namespace",
			expectedErrMsg: "unknown resource types: custom.io/v1/Custom",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()

			mapper, err := tf.ToRESTMapper()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			streamStr := ""
			for _, manifestStr := range tc.manifests {
				streamStr = streamStr + "\n---\n" + manifestStr
			}
			streamStr += "\n---\n"
			rgStreamReader := &ResourceGroupStreamManifestReader{
				ReaderName: "rgstream",
				Reader:     strings.NewReader(streamStr),
				ReaderOptions: manifestreader.ReaderOptions{
					Mapper:           mapper,
					Namespace:        tc.namespace,
					EnforceNamespace: false,
				},
			}
			readObjs, err := rgStreamReader.Read()
			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			assert.NoError(t, err)

			readObjMetas := object.UnstructuredSetToObjMetadataSet(readObjs)

			sort.Slice(readObjMetas, func(i, j int) bool {
				return readObjMetas[i].String() < readObjMetas[j].String()
			})
			assert.Equal(t, tc.expectedObjs, readObjMetas)
		})
	}
}

func TestResourceStreamManifestReader_isKptfile(t *testing.T) {
	testCases := map[string]struct {
		kptfile  string
		expected bool
	}{
		"Empty kptfile is invalid": {
			kptfile:  "",
			expected: false,
		},
		"Kptfile with foo/bar GVK is invalid": {
			kptfile: `
apiVersion: foo/v1
kind: FooBar
metadata:
  name: test1
`,
			expected: false,
		},
		"Kptfile with bad apiVersion is invalid": {
			kptfile: `
apiVersion: foo/v1
kind: Kptfile
metadata:
  name: test1
`,
			expected: false,
		},
		"Kptfile with wrong kind is invalid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: foo
metadata:
  name: test1
`,
			expected: false,
		},
		"Kptfile with different GVK is invalid": {
			kptfile: `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: test-deployment
spec:
  replicas: 1
`,
			expected: false,
		},
		"Wrong fields (foo/bar) in kptfile is invalid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: Kptfile
foo: bar
`,
			expected: false,
		},
		"Kptfile with deployment/replicas fields is invalid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-deployment
spec:
  replicas: 1
`,
			expected: false,
		},
		"Wrong fields (foo/bar) in kptfile inventory is invalid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
inventory:
  namespace: test-namespace
  name: inventory-obj-name
  foo: bar
`,
			expected: false,
		},
		"Full, regular kptfile is valid": {
			kptfile:  kptFile,
			expected: true,
		},
		"Kptfile with only GVK is valid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: Kptfile
`,
			expected: true,
		},
		"Kptfile missing optional inventory is still valid": {
			kptfile: `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
`,
			expected: true,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual := isKptfile([]byte(tc.kptfile))
			if tc.expected != actual {
				t.Errorf("expected isKptfile (%t), got (%t)", tc.expected, actual)
			}
		})
	}
}
