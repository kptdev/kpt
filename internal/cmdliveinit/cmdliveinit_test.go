// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package cmdliveinit

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
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
    name: foo
    namespace: test-namespace
    inventoryID: ` + testInventoryID + "\n"

var testTime = time.Unix(5555555, 66666666)

func TestCmd_generateID(t *testing.T) {
	testCases := map[string]struct {
		namespace string
		name      string
		t         time.Time
		expected  string
		isError   bool
	}{
		"Empty inventory namespace is an error": {
			name:      inventoryName,
			namespace: "",
			t:         testTime,
			isError:   true,
		},
		"Empty inventory name is an error": {
			name:      "",
			namespace: inventoryNamespace,
			t:         testTime,
			isError:   true,
		},
		"Namespace/name hash is valid": {
			name:      inventoryName,
			namespace: inventoryNamespace,
			t:         testTime,
			expected:  "fa6dc0d39b0465b90f101c2ad50d50e9b4022f23-5555555066666666",
			isError:   false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual, err := generateID(tc.namespace, tc.name, tc.t)
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
			namespace:       "testns",
			inventoryID:     "",
			expectAutoGenID: true,
			expectedInventory: kptfilev1.Inventory{
				Namespace:   "testns",
				Name:        "inventory-*",
				InventoryID: "33ee4887f9638ef63efe71a9a9a632d3e9e2488e-*",
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
		"Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set for package",
		},
		"The force flag allows changing inventory information even if already set": {
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

			// Otherwise, validate the kptfile values
			assert.NoError(t, err)
			kf, err := pkg.ReadKptfile(w.WorkspaceDirectory)
			assert.NoError(t, err)
			if !assert.NotNil(t, kf.Inventory) {
				t.FailNow()
			}
			actualInv := *kf.Inventory
			expectedInv := tc.expectedInventory
			assertInventoryName(t, expectedInv.Name, actualInv.Name)
			assert.Equal(t, expectedInv.Namespace, actualInv.Namespace)
			if tc.expectAutoGenID {
				assertGenInvID(t, actualInv.Name, actualInv.Namespace, actualInv.InventoryID)
			} else {
				assert.Equal(t, expectedInv.InventoryID, actualInv.InventoryID)
			}
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

func assertGenInvID(t *testing.T, name, namespace, actual string) bool {
	re := regexp.MustCompile(`^([a-z0-9]+)-[0-9]+$`)
	match := re.FindStringSubmatch(actual)
	if len(match) != 2 {
		t.Errorf("unexpected format for autogenerated inventoryID")
		return false
	}
	prefix, err := generateHash(namespace, name)
	if err != nil {
		panic(err)
	}
	if got, want := match[1], prefix; got != want {
		t.Errorf("expected prefix %q, but found %q", want, got)
		return false
	}
	return true
}
