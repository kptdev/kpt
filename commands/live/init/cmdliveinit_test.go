// Copyright 2020,2026 The kpt Authors
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

package init

import (
	"os"
	"path/filepath"
	"testing"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	rgfilev1alpha1 "github.com/kptdev/kpt/api/resourcegroup/v1alpha1"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

var (
	inventoryName      = "inventory-obj-name"
	inventoryNamespace = "test-namespace"
	inventoryID        = "XXXXXXX-OOOOOOOOOO-XXXX"
)

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

const testInventoryID = "SSSSSSSSSS-RRRRR"

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

var resourceGroupInventory = `
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: foo
  namespace: test-namespace
`

func TestCmd_generateID(t *testing.T) {
	testCases := map[string]struct {
		namespace string
		name      string
		expected  string
		isError   bool
	}{
		"Empty inventory namespace is an error": {
			name:      inventoryName,
			namespace: "",
			isError:   true,
		},
		"Empty inventory name is an error": {
			name:      "",
			namespace: inventoryNamespace,
			isError:   true,
		},
		"Namespace/name hash is valid": {
			name:      inventoryName,
			namespace: inventoryNamespace,
			expected:  "fa6dc0d39b0465b90f101c2ad50d50e9b4022f23",
			isError:   false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual, err := generateID(tc.namespace, tc.name)
			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			assert.NoError(t, err)
			if tc.expected != actual {
				t.Errorf("expecting generated id (%s), got (%s)", tc.expected, actual)
			}
		})
	}
}

func TestCmd_Run(t *testing.T) {
	testCases := map[string]struct {
		kptfile           string
		resourcegroup     string
		rgfilename        string
		name              string
		namespace         string
		inventoryID       string
		force             bool
		expectedErrorMsg  string
		expectAutoGenID   bool
		expectedInventory kptfilev1.Inventory
	}{
		"Fields are defaulted if not provided": {
			kptfile:         kptFile,
			name:            "",
			rgfilename:      "resourcegroup.yaml",
			namespace:       "testns",
			inventoryID:     "",
			expectAutoGenID: true,
			expectedInventory: kptfilev1.Inventory{
				Namespace: "testns",
			},
		},
		"Provided values are used": {
			kptfile:     kptFile,
			rgfilename:  "custom-rg.yaml",
			name:        "my-pkg",
			namespace:   "my-ns",
			inventoryID: "my-inv-id",
			expectedInventory: kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-pkg",
				InventoryID: "my-inv-id",
			},
		},
		"Provided values are used with custom resourcegroup filename": {
			kptfile:     kptFile,
			rgfilename:  "custom-rg.yaml",
			name:        "my-pkg",
			namespace:   "my-ns",
			inventoryID: "my-inv-id",
			expectedInventory: kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-pkg",
				InventoryID: "my-inv-id",
			},
		},
		"Invalid custom name is an error": {
			kptfile:          kptFile,
			name:             "INVALID_NAME!",
			rgfilename:       "resourcegroup.yaml",
			namespace:        "testns",
			expectedErrorMsg: "inventory name \"INVALID_NAME!\" is not a valid Kubernetes resource name",
		},
		"Invalid custom inventory-id is an error": {
			kptfile:          kptFile,
			name:             "my-pkg",
			inventoryID:      "-invalid-label-",
			rgfilename:       "resourcegroup.yaml",
			namespace:        "testns",
			expectedErrorMsg: "inventory-id \"-invalid-label-\" is not a valid Kubernetes label value",
		},
		"Whitespace inventory-id is an error": {
			kptfile:          kptFile,
			name:             "my-pkg",
			inventoryID:      "   ",
			rgfilename:       "resourcegroup.yaml",
			namespace:        "testns",
			expectedErrorMsg: "inventory-id must not be empty",
		},
		"Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			name:             inventoryName,
			rgfilename:       "custom-rg.yaml",
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"ResourceGroup with inventory already set is error": {
			kptfile:          kptFile,
			resourcegroup:    resourceGroupInventory,
			rgfilename:       "resourcegroup.yaml",
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set for package",
		},
		"ResourceGroup with inventory and Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			resourcegroup:    resourceGroupInventory,
			rgfilename:       "resourcegroup.yaml",
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"The force flag allows changing inventory information even if already set in Kptfile": {
			kptfile:     kptFileWithInventory,
			name:        inventoryName,
			rgfilename:  "resourcegroup.yaml",
			namespace:   inventoryNamespace,
			inventoryID: inventoryID,
			force:       true,
			expectedInventory: kptfilev1.Inventory{
				Namespace:   inventoryNamespace,
				Name:        inventoryName,
				InventoryID: inventoryID,
			},
		},
		"The force flag allows changing inventory information even if already set in ResourceGroup": {
			kptfile:       kptFile,
			resourcegroup: resourceGroupInventory,
			rgfilename:    "resourcegroup.yaml",
			name:          inventoryName,
			namespace:     inventoryNamespace,
			inventoryID:   inventoryID,
			force:         true,
			expectedInventory: kptfilev1.Inventory{
				Namespace:   inventoryNamespace,
				Name:        inventoryName,
				InventoryID: inventoryID,
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Set up fake test factory
			tf := cmdtesting.NewTestFactory().WithNamespace(tc.namespace)
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			w, clean := testutil.SetupWorkspace(t)
			defer clean()
			err := os.WriteFile(filepath.Join(w.WorkspaceDirectory, kptfilev1.KptFileName),
				[]byte(tc.kptfile), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// Create ResourceGroup file if testing the STDIN feature.
			if tc.resourcegroup != "" && tc.rgfilename != "" {
				err := os.WriteFile(filepath.Join(w.WorkspaceDirectory, tc.rgfilename),
					[]byte(tc.resourcegroup), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			revert := testutil.Chdir(t, w.WorkspaceDirectory)
			defer revert()

			runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
			runner.RGFileName = tc.rgfilename
			args := []string{}
			if tc.name != "" {
				args = append(args, "--name", tc.name)
			}
			if tc.inventoryID != "" {
				args = append(args, "--inventory-id", tc.inventoryID)
			}
			if tc.force {
				args = append(args, "--force")
			}
			runner.Command.SetArgs(args)

			err = runner.Command.Execute()

			// Check if there should be an error
			if tc.expectedErrorMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				return
			}

			// Otherwise, validate the kptfile values and/or resourcegroup values.
			var actualInv kptfilev1.Inventory
			assert.NoError(t, err)
			kf, err := kptfileutil.ReadKptfile(filesys.FileSystemOrOnDisk{}, w.WorkspaceDirectory)
			assert.NoError(t, err)

			switch tc.rgfilename {
			case "":
				if !assert.NotNil(t, kf.Inventory) {
					t.FailNow()
				}
				actualInv = *kf.Inventory
			default:
				// Check resourcegroup file if testing the STDIN feature.
				rg, err := pkg.ReadRGFile(w.WorkspaceDirectory, tc.rgfilename)
				assert.NoError(t, err)
				if !assert.NotNil(t, rg) {
					t.FailNow()
				}

				// Convert resourcegroup inventory back to Kptfile structure so we can share assertion
				// logic for Kptfile inventory and ResourceGroup inventory structure.
				actualInv = kptfilev1.Inventory{
					Name:        rg.Name,
					Namespace:   rg.Namespace,
					InventoryID: rg.Labels[rgfilev1alpha1.RGInventoryIDLabel],
				}
			}

			expectedInv := tc.expectedInventory
			expectedName := expectedInv.Name
			if expectedName == "" {
				expectedName = filepath.Base(w.WorkspaceDirectory)
			}
			assert.Equal(t, expectedName, actualInv.Name)
			assert.Equal(t, expectedInv.Namespace, actualInv.Namespace)
			if tc.expectAutoGenID {
				assertGenInvID(t, actualInv.Name, actualInv.Namespace, actualInv.InventoryID)
			} else {
				assert.Equal(t, expectedInv.InventoryID, actualInv.InventoryID)
			}
		})
	}
}

func TestCmd_Run_InvalidDirectoryName(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("testns")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	tempDir := t.TempDir()
	invalidDir := filepath.Join(tempDir, "Invalid_Dir_Name")
	err := os.MkdirAll(invalidDir, 0700)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(invalidDir, kptfilev1.KptFileName), []byte(kptFile), 0600)
	assert.NoError(t, err)

	runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner.Command.SetArgs([]string{invalidDir})
	err = runner.Command.Execute()

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "package directory name \"Invalid_Dir_Name\" is not a valid Kubernetes resource name")
		assert.Contains(t, err.Error(), "please provide a valid name using the --name flag")
	}
}

func TestCmd_Run_Deterministic(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("testns")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	w, clean := testutil.SetupWorkspace(t)
	defer clean()

	err := os.WriteFile(filepath.Join(w.WorkspaceDirectory, kptfilev1.KptFileName), []byte(kptFile), 0600)
	assert.NoError(t, err)

	revert := testutil.Chdir(t, w.WorkspaceDirectory)
	defer revert()

	// First init
	runner1 := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner1.Command.SetArgs([]string{})
	err = runner1.Command.Execute()
	assert.NoError(t, err)

	rg1, err := pkg.ReadRGFile(w.WorkspaceDirectory, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)
	expectedName := filepath.Base(w.WorkspaceDirectory)
	assert.Equal(t, expectedName, rg1.Name)
	expectedID, err := generateHash("testns", expectedName)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, rg1.Labels[rgfilev1alpha1.RGInventoryIDLabel])

	// Second init with --force simulates re-initializing the package after changes or deletion
	runner2 := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner2.Command.SetArgs([]string{"--force"})
	err = runner2.Command.Execute()
	assert.NoError(t, err)

	rg2, err := pkg.ReadRGFile(w.WorkspaceDirectory, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)
	assert.Equal(t, rg1.Name, rg2.Name)
	assert.Equal(t, rg1.Labels[rgfilev1alpha1.RGInventoryIDLabel], rg2.Labels[rgfilev1alpha1.RGInventoryIDLabel])
}

func TestCmd_Run_DeterministicAcrossFreshDirectories(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("testns")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	tempBase := t.TempDir()
	dir1 := filepath.Join(tempBase, "fresh-pkg-1")
	dir2 := filepath.Join(tempBase, "fresh-pkg-2")
	assert.NoError(t, os.MkdirAll(dir1, 0700))
	assert.NoError(t, os.MkdirAll(dir2, 0700))
	assert.NoError(t, os.WriteFile(filepath.Join(dir1, kptfilev1.KptFileName), []byte(kptFile), 0600))
	assert.NoError(t, os.WriteFile(filepath.Join(dir2, kptfilev1.KptFileName), []byte(kptFile), 0600))

	// Init dir1 with explicit name "shared-package"
	runner1 := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner1.Command.SetArgs([]string{dir1, "--name", "shared-package"})
	assert.NoError(t, runner1.Command.Execute())

	// Init dir2 with the same name "shared-package"
	runner2 := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner2.Command.SetArgs([]string{dir2, "--name", "shared-package"})
	assert.NoError(t, runner2.Command.Execute())

	rg1, err := pkg.ReadRGFile(dir1, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)
	rg2, err := pkg.ReadRGFile(dir2, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)

	assert.Equal(t, rg1.Name, rg2.Name)
	assert.NotEmpty(t, rg1.Labels[rgfilev1alpha1.RGInventoryIDLabel])
	assert.Equal(t, rg1.Labels[rgfilev1alpha1.RGInventoryIDLabel], rg2.Labels[rgfilev1alpha1.RGInventoryIDLabel])
}

func TestCmd_Run_DifferentNameOrNamespace(t *testing.T) {
	tf1 := cmdtesting.NewTestFactory().WithNamespace("ns-one")
	defer tf1.Cleanup()
	tf2 := cmdtesting.NewTestFactory().WithNamespace("ns-two")
	defer tf2.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	tempBase := t.TempDir()
	pkgA := filepath.Join(tempBase, "pkg-a")
	pkgB := filepath.Join(tempBase, "pkg-b")
	pkgC := filepath.Join(tempBase, "pkg-c")
	assert.NoError(t, os.MkdirAll(pkgA, 0700))
	assert.NoError(t, os.MkdirAll(pkgB, 0700))
	assert.NoError(t, os.MkdirAll(pkgC, 0700))
	assert.NoError(t, os.WriteFile(filepath.Join(pkgA, kptfilev1.KptFileName), []byte(kptFile), 0600))
	assert.NoError(t, os.WriteFile(filepath.Join(pkgB, kptfilev1.KptFileName), []byte(kptFile), 0600))
	assert.NoError(t, os.WriteFile(filepath.Join(pkgC, kptfilev1.KptFileName), []byte(kptFile), 0600))

	// pkgA: name "package-a", namespace "ns-one"
	rA := NewRunner(fake.CtxWithDefaultPrinter(), tf1, ioStreams)
	rA.Command.SetArgs([]string{pkgA, "--name", "package-a"})
	assert.NoError(t, rA.Command.Execute())

	// pkgB: different name "package-b", same namespace "ns-one"
	rB := NewRunner(fake.CtxWithDefaultPrinter(), tf1, ioStreams)
	rB.Command.SetArgs([]string{pkgB, "--name", "package-b"})
	assert.NoError(t, rB.Command.Execute())

	// pkgC: same name "package-a", different namespace "ns-two"
	rC := NewRunner(fake.CtxWithDefaultPrinter(), tf2, ioStreams)
	rC.Command.SetArgs([]string{pkgC, "--name", "package-a"})
	assert.NoError(t, rC.Command.Execute())

	rgA, err := pkg.ReadRGFile(pkgA, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)
	rgB, err := pkg.ReadRGFile(pkgB, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)
	rgC, err := pkg.ReadRGFile(pkgC, rgfilev1alpha1.RGFileName)
	assert.NoError(t, err)

	idA := rgA.Labels[rgfilev1alpha1.RGInventoryIDLabel]
	idB := rgB.Labels[rgfilev1alpha1.RGInventoryIDLabel]
	idC := rgC.Labels[rgfilev1alpha1.RGInventoryIDLabel]

	assert.NotEqual(t, idA, idB, "different names in same namespace should produce different inventory IDs")
	assert.NotEqual(t, idA, idC, "same name in different namespaces should produce different inventory IDs")
}

func assertGenInvID(t *testing.T, name, namespace, actual string) bool {
	expected, err := generateHash(namespace, name)
	if !assert.NoError(t, err) {
		return false
	}
	return assert.Equal(t, expected, actual)
}
