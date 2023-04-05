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

package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var testNamespace = "test-inventory-namespace"
var inventoryObjName = "test-inventory-obj"
var testInventoryLabel = "test-inventory-label"

var rgInvObj = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "kpt.dev/v1alpha1",
		"kind":       "ResourceGroup",
		"metadata": map[string]interface{}{
			"name":      inventoryObjName,
			"namespace": testNamespace,
			"labels": map[string]interface{}{
				common.InventoryLabel: testInventoryLabel,
			},
		},
		"spec": map[string]interface{}{
			"resources": []interface{}{},
		},
	},
}

var cmInvStr = `
kind: ConfigMap
apiVersion: v1
metadata:
  name:      test-inventory-obj
  namespace: test-inventory-namespace
  labels:
    cli-utils.sigs.k8s.io/inventory-id: test-inventory-label
`

var cmInvObj = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      inventoryObjName,
			"namespace": testNamespace,
			"labels": map[string]interface{}{
				common.InventoryLabel: testInventoryLabel,
			},
		},
	},
}

var pod1 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":      "pod-1",
			"namespace": testNamespace,
		},
	},
}

var pod2 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":      "pod-2",
			"namespace": testNamespace,
		},
	},
}

func TestKptMigrate_migrateKptfileToRG(t *testing.T) {
	testCases := map[string]struct {
		kptfile       string
		rgFilename    string
		resourcegroup string
		dryRun        bool
		isError       bool
	}{
		"Missing Kptfile is an error": {
			kptfile:    "",
			rgFilename: "resourcegroup.yaml",
			dryRun:     false,
			isError:    true,
		},
		"Kptfile with existing inventory will create ResourceGroup": {
			kptfile:    kptFileWithInventory,
			rgFilename: "resourcegroup.yaml",
			dryRun:     false,
			isError:    false,
		},
		"ResopurceGroup file already exists will error": {
			kptfile:       kptFileWithInventory,
			rgFilename:    "resourcegroup.yaml",
			resourcegroup: resourceGroupInventory,
			dryRun:        false,
			isError:       true,
		},
		"Dry-run will not fill in inventory fields": {
			kptfile:    kptFile,
			rgFilename: "resourcegroup.yaml",
			dryRun:     true,
			isError:    false,
		},
		"Custom ResourceGroup file will be generated": {
			kptfile:    kptFileWithInventory,
			rgFilename: "custom-rg.yaml",
			dryRun:     false,
			isError:    false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Set up fake test factory
			tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			// Set up temp directory with Ktpfile
			dir := t.TempDir()
			p := filepath.Join(dir, "Kptfile")
			err := os.WriteFile(p, []byte(tc.kptfile), 0600)
			assert.NoError(t, err)

			if tc.resourcegroup != "" {
				p := filepath.Join(dir, tc.rgFilename)
				err = os.WriteFile(p, []byte(tc.resourcegroup), 0600)
				assert.NoError(t, err)
			}

			ctx := fake.CtxWithDefaultPrinter()
			// Create MigrateRunner and call "updateKptfile"
			cmLoader := manifestreader.NewManifestLoader(tf)
			migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
			migrateRunner.dryRun = tc.dryRun
			migrateRunner.rgFile = tc.rgFilename
			migrateRunner.cmInvClientFunc = func(factory util.Factory) (inventory.Client, error) {
				return inventory.NewFakeClient([]object.ObjMetadata{}), nil
			}
			err = migrateRunner.migrateKptfileToRG([]string{dir})
			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			rg, err := pkg.ReadRGFile(dir, migrateRunner.rgFile)
			if !tc.dryRun && !assert.NoError(t, err) {
				t.FailNow()
			}

			// Ensure the Kptfile does not contain inventory information.
			if !assert.Nil(t, kf.Inventory) {
				t.Errorf("inventory information should not be set in Kptfile")
			}

			if !tc.dryRun {
				if rg == nil {
					t.Fatalf("unable to read ResourceGroup file")
				}
				assert.Equal(t, inventoryNamespace, rg.ObjectMeta.Namespace)
				if len(rg.ObjectMeta.Name) == 0 {
					t.Errorf("inventory name not set in Kptfile")
				}
				if rg.ObjectMeta.Labels[rgfilev1alpha1.RGInventoryIDLabel] != testInventoryID {
					t.Errorf("inventory id not set correctly in ResourceGroup: %s", rg.ObjectMeta.Labels[rgfilev1alpha1.RGInventoryIDLabel])
				}
			} else if rg != nil {
				t.Errorf("inventory shouldn't be set during dryrun")
			}
		})
	}
}

func TestKptMigrate_retrieveConfigMapInv(t *testing.T) {
	testCases := map[string]struct {
		configMap string
		expected  *unstructured.Unstructured
		isError   bool
	}{
		"Missing ConfigMap is an error": {
			configMap: "",
			expected:  nil,
			isError:   true,
		},
		"ConfigMap inventory object is correctly retrieved": {
			configMap: cmInvStr,
			expected:  cmInvObj,
			isError:   false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Set up fake test factory
			tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			ctx := fake.CtxWithDefaultPrinter()
			// Create MigrateRunner and call "retrieveConfigMapInv"
			cmLoader := manifestreader.NewManifestLoader(tf)
			migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
			migrateRunner.cmInvClientFunc = func(factory util.Factory) (inventory.Client, error) {
				return inventory.NewFakeClient([]object.ObjMetadata{}), nil
			}
			actual, err := migrateRunner.retrieveConfigMapInv(strings.NewReader(tc.configMap), []string{"-"})
			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			if tc.expected.GetName() != actual.Name() {
				t.Errorf("expected ConfigMap (%#v), got (%#v)", tc.expected, actual)
			}
			if tc.expected.GetNamespace() != actual.Namespace() {
				t.Errorf("expected ConfigMap (%#v), got (%#v)", tc.expected, actual)
			}
		})
	}
}

func TestKptMigrate_findResourceGroupInv(t *testing.T) {
	testCases := map[string]struct {
		objs     []*unstructured.Unstructured
		expected *unstructured.Unstructured
		isError  bool
	}{
		"Empty objs returns an error": {
			objs:     []*unstructured.Unstructured{},
			expected: nil,
			isError:  true,
		},
		"Objs without inventory obj returns an error": {
			objs:     []*unstructured.Unstructured{pod1},
			expected: nil,
			isError:  true,
		},
		"Objs without ConfigMap inventory obj returns an error": {
			objs:     []*unstructured.Unstructured{cmInvObj, pod1},
			expected: nil,
			isError:  true,
		},
		"Objs without ResourceGroup inventory obj returns ResourceGroup": {
			objs:     []*unstructured.Unstructured{rgInvObj, pod1},
			expected: rgInvObj,
			isError:  false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual, err := findResourceGroupInv(tc.objs)
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			if tc.expected != actual {
				t.Errorf("expected ResourceGroup (%#v), got (%#v)", tc.expected, actual)
			}
		})
	}
}

func TestKptMigrate_migrateObjs(t *testing.T) {
	testCases := map[string]struct {
		invObj     string
		objs       []object.ObjMetadata
		isError    bool
		rgFilename string
	}{
		"No objects to migrate is valid": {
			invObj:  "",
			objs:    []object.ObjMetadata{},
			isError: false,
		},
		"One migrate object is valid": {
			invObj:     resourceGroupInventory,
			objs:       []object.ObjMetadata{object.UnstructuredToObjMetadata(pod1)},
			rgFilename: rgfilev1alpha1.RGFileName,
			isError:    false,
		},
		"Multiple migrate objects are valid": {
			invObj: resourceGroupInventory,
			objs: []object.ObjMetadata{
				object.UnstructuredToObjMetadata(pod1),
				object.UnstructuredToObjMetadata(pod2),
			},
			rgFilename: rgfilev1alpha1.RGFileName,
			isError:    false,
		},
		"Kptfile does not have inventory is valid": {
			invObj:     resourceGroupInventory,
			objs:       []object.ObjMetadata{},
			rgFilename: rgfilev1alpha1.RGFileName,
			isError:    false,
		},
		"One migrate object is valid with inventory in Kptfile": {
			invObj:     resourceGroupInventory,
			objs:       []object.ObjMetadata{object.UnstructuredToObjMetadata(pod1)},
			rgFilename: rgfilev1alpha1.RGFileName,
			isError:    false,
		},
		"Migrate to ResourceGroup object with custom filename": {
			invObj:     resourceGroupInventory,
			objs:       []object.ObjMetadata{object.UnstructuredToObjMetadata(pod1)},
			rgFilename: "test-rg.yaml",
			isError:    false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Set up fake test factory
			tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			ctx := fake.CtxWithDefaultPrinter()
			// Create MigrateRunner and call "retrieveConfigMapInv"
			rgInvClient := inventory.NewFakeClient(tc.objs)
			cmLoader := manifestreader.NewManifestLoader(tf)
			migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
			migrateRunner.rgFile = tc.rgFilename

			err := migrateRunner.migrateObjs(rgInvClient, tc.objs, strings.NewReader(tc.invObj), []string{"-"})
			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			// Retrieve the objects stored by the inventory client and validate.
			migratedObjs, err := rgInvClient.GetClusterObjs(nil)
			assert.NoError(t, err)
			if len(tc.objs) != len(migratedObjs) {
				t.Errorf("expected num migrated objs (%d), got (%d)", len(tc.objs), len(migratedObjs))
			}
			for _, migratedObj := range migratedObjs {
				found := false
				var expectedObj object.ObjMetadata
				for _, expectedObj = range tc.objs {
					if expectedObj == migratedObj {
						found = true
					}
				}
				if !found {
					t.Fatalf("expected migrated object (%#v), but not found", expectedObj)
					return
				}
			}
		})
	}
}

var kptFileWithInventory = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
upstreamLock:
  type: git
  git:
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
inventory:
    name: foo
    namespace: test-namespace
    inventoryID: ` + testInventoryID + "\n"

const testInventoryID = "SSSSSSSSSS-RRRRR"

var kptFile = `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
upstreamLock:
  type: git
  git:
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
`

var inventoryNamespace = "test-namespace"

var resourceGroupInventory = `
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: foo
  namespace: test-namespace
  labels:
    cli-utils.sigs.k8s.io/inventory-id: SSSSSSSSSS-RRRRR
`
