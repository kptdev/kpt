// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

func TestResourceStreamManifestReader_Read(t *testing.T) {
	testCases := map[string]struct {
		manifests map[string]string
		numObjs   int
	}{
		"Kptfile only is valid": {
			manifests: map[string]string{
				"Kptfile": kptFile,
			},
			numObjs: 1,
		},
		"Only a pod is valid": {
			manifests: map[string]string{
				"pod-a.yaml": podA,
			},
			numObjs: 1,
		},
		"Multiple pods are valid": {
			manifests: map[string]string{
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numObjs: 2,
		},
		"Basic ResourceGroup inventory object created": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
				"pod-a.yaml": podA,
			},
			numObjs: 2,
		},
		"ResourceGroup inventory object created, multiple objects": {
			manifests: map[string]string{
				"Kptfile":           kptFile,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numObjs: 3,
		},
		"ResourceGroup inventory object created, Kptfile last": {
			manifests: map[string]string{
				"deployment-a.yaml": deploymentA,
				"Kptfile":           kptFile,
			},
			numObjs: 2,
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
			streamReader := &manifestreader.StreamManifestReader{
				ReaderName: "rgstream",
				Reader:     strings.NewReader(streamStr),
				ReaderOptions: manifestreader.ReaderOptions{
					Mapper:           mapper,
					Namespace:        inventoryNamespace,
					EnforceNamespace: false,
				},
			}
			rgStreamReader := &ResourceGroupStreamManifestReader{
				streamReader: streamReader,
			}
			readObjs, err := rgStreamReader.Read()
			assert.NoError(t, err)
			assert.Equal(t, tc.numObjs, len(readObjs))
			for _, obj := range readObjs {
				assert.Equal(t, inventoryNamespace, obj.GetNamespace())
			}
			invObj := findResourceGroupInventory(readObjs)
			if invObj != nil {
				assert.Equal(t, inventoryName, invObj.GetName())
				actualID, err := getInventoryLabel(invObj)
				assert.NoError(t, err)
				assert.Equal(t, inventoryID, actualID)
			}
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
apiVersion: kpt.dev/v1alpha1
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
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
foo: bar
`,
			expected: false,
		},
		"Kptfile with deployment/replicas fields is invalid": {
			kptfile: `
apiVersion: kpt.dev/v1alpha1
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
apiVersion: kpt.dev/v1alpha1
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
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
`,
			expected: true,
		},
		"Kptfile missing optional inventory is still valid": {
			kptfile: `
apiVersion: kpt.dev/v1alpha1
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

// Returns the ResourceGroup inventory object from a slice
// of objects, or nil if it does not exist.
func findResourceGroupInventory(infos []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, info := range infos {
		invLabel, _ := getInventoryLabel(info)
		if len(invLabel) != 0 {
			return info
		}
	}
	return nil
}
