// Copyright 2020, 2026 The kpt Authors
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
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rgfilev1alpha1 "github.com/kptdev/kpt/api/resourcegroup/v1alpha1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/kptdev/kpt/pkg/printer/fake"
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
	Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      inventoryObjName,
			"namespace": testNamespace,
			"labels": map[string]any{
				common.InventoryLabel: testInventoryLabel,
			},
		},
	},
}

var pod1 = &unstructured.Unstructured{
	Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
			"name":      "pod-1",
			"namespace": testNamespace,
		},
	},
}

var pod2 = &unstructured.Unstructured{
	Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]any{
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
			migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
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
			kf, err := kptfileutil.ReadKptfile(filesys.FileSystemOrOnDisk{}, dir)
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

			if tc.dryRun {
				if rg != nil {
					t.Errorf("inventory shouldn't be set during dryrun")
				}
				return
			}

			if rg == nil {
				t.Fatalf("unable to read ResourceGroup file")
				return
			}
			assert.Equal(t, inventoryNamespace, rg.Namespace)
			if len(rg.Name) == 0 {
				t.Errorf("inventory name not set in Kptfile")
			}
			if rg.Labels[rgfilev1alpha1.RGInventoryIDLabel] != testInventoryID {
				t.Errorf("inventory id not set correctly in ResourceGroup: %s", rg.Labels[rgfilev1alpha1.RGInventoryIDLabel])
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
			migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
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

func TestKptMigrate_Run_NonKRM(t *testing.T) {
	t.Run("non-KRM file with no annotation blocks Run with a consolidated, file-naming error", func(t *testing.T) {
		tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
		defer tf.Cleanup()
		ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Kptfile"), []byte(kptFile), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)

		ctx := fake.CtxWithDefaultPrinter()
		cmLoader := manifestreader.NewManifestLoader(tf)
		migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
		migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
			return inventory.NewFakeClient([]object.ObjMetadata{}), nil
		}

		err = migrateRunner.Run(strings.NewReader(""), []string{dir})
		if !assert.Error(t, err) {
			t.FailNow()
		}
		assert.Contains(t, err.Error(), "values.yaml")
		assert.Contains(t, err.Error(), "1 file(s)")
		assert.Contains(t, err.Error(), ".krmignore")
		assert.NotContains(t, err.Error(), "add `kpt.dev/auto-ignore-non-krm", "must not suggest the annotation as a fix")
	})

	t.Run("the auto-ignore annotation does NOT unblock migrate -- still fails with the same pre-scan error", func(t *testing.T) {
		tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
		defer tf.Cleanup()
		ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

		dir := t.TempDir()
		kptFileWithAnnotation := `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
  annotations:
    kpt.dev/auto-ignore-non-krm: "true"
upstreamLock:
  type: git
  git:
    repo: git@github.com:seans3/blueprint-helloworld
    directory: /
    ref: master
`
		err := os.WriteFile(filepath.Join(dir, "Kptfile"), []byte(kptFileWithAnnotation), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)

		ctx := fake.CtxWithDefaultPrinter()
		cmLoader := manifestreader.NewManifestLoader(tf)
		migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
		migrateRunner.dryRun = true
		migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
			return inventory.NewFakeClient([]object.ObjMetadata{}), nil
		}

		err = migrateRunner.Run(strings.NewReader(""), []string{dir})
		if !assert.Error(t, err) {
			t.FailNow()
		}
		assert.Contains(t, err.Error(), "values.yaml")
		assert.Contains(t, err.Error(), "1 file(s)")

		// Confirm the annotation did not cause the file to be skipped or modified.
		content, readErr := os.ReadFile(filepath.Join(dir, "values.yaml"))
		assert.NoError(t, readErr)
		assert.Equal(t, "replicaCount: 3\n", string(content))
		_, statErr := os.Stat(filepath.Join(dir, ".krmignore"))
		assert.True(t, os.IsNotExist(statErr), ".krmignore should not have been written")
	})

	t.Run("same-named non-KRM files in different subpackages are disambiguated by path", func(t *testing.T) {
		tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
		defer tf.Cleanup()
		ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

		dir := t.TempDir()
		app1 := filepath.Join(dir, "apps", "app1")
		app2 := filepath.Join(dir, "apps", "app2")
		assert.NoError(t, os.MkdirAll(app1, 0700))
		assert.NoError(t, os.MkdirAll(app2, 0700))
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "Kptfile"), []byte(kptFile), 0600))
		assert.NoError(t, os.WriteFile(filepath.Join(app1, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: app1\n"), 0600))
		assert.NoError(t, os.WriteFile(filepath.Join(app2, "Kptfile"), []byte("apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: app2\n"), 0600))
		assert.NoError(t, os.WriteFile(filepath.Join(app1, "values.yaml"), []byte("replicaCount: 1\n"), 0600))
		assert.NoError(t, os.WriteFile(filepath.Join(app2, "values.yaml"), []byte("replicaCount: 2\n"), 0600))

		ctx := fake.CtxWithDefaultPrinter()
		cmLoader := manifestreader.NewManifestLoader(tf)
		migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
		migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
			return inventory.NewFakeClient([]object.ObjMetadata{}), nil
		}

		err := migrateRunner.Run(strings.NewReader(""), []string{dir})
		if !assert.Error(t, err) {
			t.FailNow()
		}
		assert.Contains(t, err.Error(), filepath.ToSlash(filepath.Join("apps", "app1", "values.yaml")))
		assert.Contains(t, err.Error(), filepath.ToSlash(filepath.Join("apps", "app2", "values.yaml")))
		assert.Contains(t, err.Error(), "2 file(s)")
	})

	t.Run(".krmignore still works independently of the annotation, no non-KRM-file error at all", func(t *testing.T) {
		tf := cmdtesting.NewTestFactory().WithNamespace(inventoryNamespace)
		defer tf.Cleanup()
		ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "Kptfile"), []byte(kptFile), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("replicaCount: 3\n"), 0600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, ".krmignore"), []byte("values.yaml\n"), 0600)
		assert.NoError(t, err)

		ctx := fake.CtxWithDefaultPrinter()
		cmLoader := manifestreader.NewManifestLoader(tf)
		migrateRunner := NewRunner(ctx, tf, cmLoader, ioStreams)
		migrateRunner.cmInvClientFunc = func(_ context.Context, _ util.Factory) (inventory.Client, error) {
			return inventory.NewFakeClient([]object.ObjMetadata{}), nil
		}

		err = migrateRunner.Run(strings.NewReader(""), []string{dir})
		// The pre-scan must not block this case. Any other downstream error
		// (e.g. from the fake ConfigMap-based inventory lookup) is tolerated.
		if err != nil {
			assert.NotContains(t, err.Error(), "values.yaml")
			assert.NotContains(t, err.Error(), "must contain only KRM resources")
		}
	})
}
