// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
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
		manifests map[string]string
		numInfos  int
		annotated bool
		isError   bool
	}{
		"Kptfile missing inventory id is error": {
			manifests: map[string]string{
				"Kptfile":    kptFileMissingID,
				"pod-a.yaml": podA,
			},
			numInfos: 0,
			isError:  true,
		},
		"Basic ResourceGroup inventory object created": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
				"pod-a.yaml": podA,
			},
			numInfos: 2,
			isError:  false,
		},
		"ResourceGroup inventory object created, multiple objects": {
			manifests: map[string]string{
				"Kptfile":           kptFile,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numInfos: 3,
			isError:  false,
		},
		"ResourceGroup inventory object created, Kptfile last": {
			manifests: map[string]string{
				"deployment-a.yaml": deploymentA,
				"Kptfile":           kptFile,
			},
			numInfos: 2,
			isError:  false,
		},
		"ResourceGroup inventory object created with annotation, multiple objects": {
			manifests: map[string]string{
				"Kptfile":           kptFileWithAnnotations,
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numInfos: 3,
			annotated: true,
			isError:  false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()

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
					Factory:          tf,
					Namespace:        inventoryNamespace,
					EnforceNamespace: false,
				},
			}
			rgPathReader := &ResourceGroupPathManifestReader{
				pathReader: pathReader,
			}
			readInfos, err := rgPathReader.Read()

			// Validate the returned values are correct.
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, len(readInfos), tc.numInfos)
			for _, info := range readInfos {
				assert.Equal(t, inventoryNamespace, info.Namespace)
			}
			invInfo, _, err := inventory.SplitInfos(readInfos)
			assert.NoError(t, err)
			assert.Equal(t, inventoryName, invInfo.Name)
			actualID, err := getInventoryLabel(invInfo)
			assert.NoError(t, err)
			assert.Equal(t, inventoryID, actualID)
			actualAnnotations := getInventoryAnnotations(invInfo)
			if tc.annotated {
				assert.Equal(t, map[string]string{"random-key": "random-value"}, actualAnnotations)
			} else {
			  assert.Equal(t, map[string]string(nil), actualAnnotations)
			}
		})
	}
}

func getInventoryLabel(inv *resource.Info) (string, error) {
	obj := inv.Object
	if obj == nil {
		return "", fmt.Errorf("inventory object is nil")
	}
	accessor, err := meta.Accessor(obj)
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

func getInventoryAnnotations(inv *resource.Info) map[string]string {
	obj := inv.Object
	if obj == nil {
		return nil
	}
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil
	}
	return accessor.GetAnnotations()
}