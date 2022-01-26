// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdliveinit

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
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
    name: inventory-obj-name
    namespace: test-namespace
    inventoryID: ` + testInventoryID + "\n"

var resourceGroupInventory = `
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: inventory-obj-name
  namespace: test-namespace
`

func TestCmd_Run_NoKptfile(t *testing.T) {
	// Set up fake test factory
	tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	w, clean := testutil.SetupWorkspace(t)
	defer clean()

	revert := testutil.Chdir(t, w.WorkspaceDirectory)
	defer revert()

	runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner.Command.SetArgs([]string{})
	err := runner.Command.Execute()

	if !assert.Error(t, err) {
		t.FailNow()
	}
	assert.Contains(t, err.Error(), "error reading Kptfile at")
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
		expectedInventory kptfilev1.Inventory
	}{
		"Fields are defaulted if not provided": {
			kptfile:     kptFile,
			name:        "",
			namespace:   "testns",
			inventoryID: "",
			expectedInventory: kptfilev1.Inventory{
				Namespace:   "testns",
				Name:        "inventory-*",
				InventoryID: "",
			},
		},
		"Provided values are used": {
			kptfile:     kptFile,
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
		"Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"ResourceGroup with inventory already set is error": {
			kptfile:          kptFile,
			resourcegroup:    resourceGroupInventory,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set for package",
		},
		"ResourceGroup with inventory and Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			resourcegroup:    resourceGroupInventory,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"The force flag allows changing inventory information even if already set in Kptfile": {
			kptfile:     kptFileWithInventory,
			name:        inventoryName,
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
			err := ioutil.WriteFile(filepath.Join(w.WorkspaceDirectory, kptfilev1.KptFileName),
				[]byte(tc.kptfile), 0600)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			// Create ResourceGroup file if specified by test.
			rgfilename := rgfilev1alpha1.RGFileName
			if tc.rgfilename != "" {
				rgfilename = tc.rgfilename
			}
			if tc.resourcegroup != "" {
				err := ioutil.WriteFile(filepath.Join(w.WorkspaceDirectory, rgfilename),
					[]byte(tc.resourcegroup), 0600)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

			revert := testutil.Chdir(t, w.WorkspaceDirectory)
			defer revert()

			runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
			runner.namespace = tc.namespace
			args := []string{
				"--name", tc.name,
				"--inventory-id", tc.inventoryID,
			}
			if tc.force {
				args = append(args, "--force")
			}
			if tc.rgfilename != "" {
				args = append(args, "--rg-file", tc.rgfilename)
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

			// Otherwise, validate the kptfile and resourcegroup values
			assert.NoError(t, err)
			kf, err := pkg.ReadKptfile(w.WorkspaceDirectory)
			assert.NoError(t, err)
			if !assert.Nil(t, kf.Inventory) {
				t.FailNow()
			}
			assert.NoError(t, err)
			rg, err := pkg.ReadRGFile(w.WorkspaceDirectory, rgfilename)
			assert.NoError(t, err)
			if !assert.NotNil(t, rg) {
				t.FailNow()
			}

			actualInv := kptfilev1.Inventory{
				Name:        rg.Name,
				Namespace:   rg.Namespace,
				InventoryID: rg.Labels[rgfilev1alpha1.RGInventoryIDLabel],
			}
			expectedInv := tc.expectedInventory
			assertInventoryName(t, expectedInv.Name, actualInv.Name)
			assert.Equal(t, expectedInv.Namespace, actualInv.Namespace)
			assert.Equal(t, expectedInv.InventoryID, actualInv.InventoryID)
		})
	}
}

func assertInventoryName(t *testing.T, expected, actual string) bool {
	re := regexp.MustCompile(`^inventory-[0-9]+$`)
	if expected == "inventory-*" {
		if re.MatchString(actual) {
			return true
		}
		t.Errorf("expected value on the format 'inventory-[0-9]+', but found %q", actual)
	}
	return assert.Equal(t, expected, actual)
}
