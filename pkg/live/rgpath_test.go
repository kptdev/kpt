// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

var (
	inventoryNamespace = "test-namespace"
	inventoryName      = "inventory-obj-name"
	inventoryID        = "XXXXXXXXXX-FOOOOOO"
)

var kptFile = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: test1
upstream:
  type: git
  git:
    commit: 786b898857bd7e9647c229d5f39b0be4de86c915
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
inventory:
  namespace: test-namespace
  name: inventory-obj-name
  inventoryID: XXXXXXXXXX-FOOOOOO
`

var kptFileMissingID = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: test1
upstream:
  type: git
  git:
    commit: 786b898857bd7e9647c229d5f39b0be4de86c915
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
inventory:
  namespace: test-namespace
  name: inventory-obj-name
`
var kptFileWithAnnotations = `
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: test1
upstream:
  type: git
  git:
    commit: 786b898857bd7e9647c229d5f39b0be4de86c915
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
inventory:
  namespace: test-namespace
  name: inventory-obj-name
  inventoryID: XXXXXXXXXX-FOOOOOO
  annotations:
    random-key: random-value
`

var podA = `
apiVersion: v1
kind: Pod
metadata:
  name: pod-a
  namespace: test-namespace
  labels:
    name: test-pod-label
spec:
  containers:
  - name: kubernetes-pause
    image: k8s.gcr.io/pause:2.0
`

var deploymentA = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: test-deployment
spec:
  replicas: 1
`

func TestInvGenPathManifestReader_Read(t *testing.T) {
	testCases := map[string]struct {
		manifests  map[string]string
		numObjs    int
		hasKptfile bool
		annotated  bool
	}{
		"Kptfile missing inventory id returns only Pod": {
			manifests: map[string]string{
				"Kptfile":    kptFileMissingID,
				"pod-a.yaml": podA,
			},
			numObjs:    1,
			hasKptfile: false,
		},
		"Basic ResourceGroup inventory object created": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
				"pod-a.yaml": podA,
			},
			numObjs:    2,
			hasKptfile: true,
		},
		"ResourceGroup inventory object created, multiple objects": {
			manifests: map[string]string{
				"Kptfile":           kptFile,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numObjs:    3,
			hasKptfile: true,
		},
		"ResourceGroup inventory object created, Kptfile last": {
			manifests: map[string]string{
				"deployment-a.yaml": deploymentA,
				"Kptfile":           kptFile,
			},
			numObjs:    2,
			hasKptfile: true,
		},
		"ResourceGroup inventory object created with annotation, multiple objects": {
			manifests: map[string]string{
				"Kptfile":           kptFileWithAnnotations,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numObjs:    3,
			hasKptfile: true,
			annotated:  true,
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
			// the manifests into infos.
			pathReader := &manifestreader.PathManifestReader{
				Path: dir,
				ReaderOptions: manifestreader.ReaderOptions{
					Mapper:           mapper,
					Namespace:        inventoryNamespace,
					EnforceNamespace: false,
				},
			}
			rgPathReader := &ResourceGroupPathManifestReader{
				pathReader: pathReader,
			}
			readObjs, err := rgPathReader.Read()
			assert.NoError(t, err)
			assert.Equal(t, len(readObjs), tc.numObjs)
			for _, obj := range readObjs {
				assert.Equal(t, inventoryNamespace, obj.GetNamespace())
			}
			if tc.hasKptfile {
				invObj, _, err := inventory.SplitUnstructureds(readObjs)
				assert.NoError(t, err)
				assert.Equal(t, inventoryName, invObj.GetName())
				actualID, err := getInventoryLabel(invObj)
				assert.NoError(t, err)
				assert.Equal(t, inventoryID, actualID)
				actualAnnotations := getInventoryAnnotations(invObj)
				if tc.annotated {
					assert.Equal(t, map[string]string{"random-key": "random-value"}, actualAnnotations)
				} else {
					assert.Equal(t, map[string]string(nil), actualAnnotations)
				}
			}
		})
	}
}

func getInventoryLabel(inv *unstructured.Unstructured) (string, error) {
	accessor, err := meta.Accessor(inv)
	if err != nil {
		return "", err
	}
	labels := accessor.GetLabels()
	inventoryLabel, exists := labels[common.InventoryLabel]
	if !exists {
		return "", fmt.Errorf("inventory label does not exist for inventory object: %s", common.InventoryLabel)
	}
	return inventoryLabel, nil
}

func getInventoryAnnotations(inv *unstructured.Unstructured) map[string]string {
	accessor, err := meta.Accessor(inv)
	if err != nil {
		return nil
	}
	return accessor.GetAnnotations()
}

func TestResourceGroupUnstructured(t *testing.T) {
	name := "name"
	namespace := "test"
	id := "random-id"
	rg := ResourceGroupUnstructured(name, namespace, id)
	if rg == nil {
		t.Fatal("resourcegroup shouldn't be nil")
	}
	if rg.GetName() != name {
		t.Fatalf("resourcegroup name expected %s, but got %s", name, rg.GetName())
	}
	if rg.GetNamespace() != namespace {
		t.Fatalf("resourcegroup namespace expected %s, but got %s", namespace, rg.GetNamespace())
	}
	if rg.GetLabels()[common.InventoryLabel] != id {
		t.Fatalf("resourcegroup inventory id expected %s, but got %s", id, rg.GetLabels()[common.InventoryLabel])
	}
}
