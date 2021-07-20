// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestPathManifestReader_Read(t *testing.T) {
	_ = apiextv1.AddToScheme(scheme.Scheme)
	testCases := map[string]struct {
		manifests      map[string]string
		namespace      string
		expectedObjs   []object.ObjMetadata
		expectedErrMsg string
	}{
		"Empty package is ok": {
			manifests:    map[string]string{},
			namespace:    "test-namespace",
			expectedObjs: []object.ObjMetadata{},
		},
		"Kptfile are ignored": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
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
		"Namespace gets set on namespaced resources": {
			manifests: map[string]string{
				"pod-a.yaml":      podA,
				"deployment.yaml": deploymentA,
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
		"Function config resources are ignored": {
			manifests: map[string]string{
				"Kptfile":           kptFileWithPipeline,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
				"cm.yaml":           configMap,
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
		"Function config resources which are marked as not being local config remains": {
			manifests: map[string]string{
				"Kptfile":           kptFileWithPipeline,
				"deployment-a.yaml": deploymentA,
				"cm.yaml":           notLocalConfig,
			},
			namespace: "test-namespace",
			expectedObjs: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "",
						Kind:  "ConfigMap",
					},
					Name:      "cm",
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
			expectedErrMsg: "unknown resource types: Custom.custom.io",
		},
		"local-config is filtered out": {
			manifests: map[string]string{
				"deployment-a.yaml": deploymentA,
				"lc.yaml":           localConfig,
			},
			namespace: "test-namespace",
			expectedObjs: []object.ObjMetadata{
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
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()

			mapper, err := tf.ToRESTMapper()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// Set up the yaml manifests (including Kptfile) in temp dir.
			dir, err := ioutil.TempDir("", "path-reader-test")
			assert.NoError(t, err)
			for filename, content := range tc.manifests {
				p := filepath.Join(dir, filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}

			// Create the ResourceGroupPathManifestReader, and Read()
			// the manifests into unstructureds
			rgPathReader := &ResourceGroupPathManifestReader{
				PkgPath: dir,
				ReaderOptions: manifestreader.ReaderOptions{
					Mapper:           mapper,
					Namespace:        tc.namespace,
					EnforceNamespace: false,
				},
			}
			readObjs, err := rgPathReader.Read()
			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			assert.NoError(t, err)

			readObjMetas := object.UnstructuredsToObjMetasOrDie(readObjs)

			sort.Slice(readObjMetas, func(i, j int) bool {
				return readObjMetas[i].String() < readObjMetas[j].String()
			})
			assert.Equal(t, tc.expectedObjs, readObjMetas)
		})
	}
}
