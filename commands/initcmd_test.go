// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	inventoryName      = "inventory-obj-name"
	inventoryNamespace = "test-namespace"
	inventoryID        = "XXXXXXX-OOOOOOOOOO-XXXX"
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
`

var kptFileWithInventory = `
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
    name: foo
    namespace: test-namespace
    inventoryID: SSSSSSSSSS-RRRRR
`

var testTime = time.Unix(5555555, 66666666)

func TestKptInitOptions_generateID(t *testing.T) {
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

func TestKptInitOptions_updateKptfile(t *testing.T) {
	testCases := map[string]struct {
		kptfile     string
		name        string
		namespace   string
		inventoryID string
		force       bool
		isError     bool
	}{
		"Empty inventory name is an error": {
			kptfile:     kptFile,
			name:        "",
			namespace:   inventoryNamespace,
			inventoryID: inventoryID,
			force:       false,
			isError:     true,
		},
		"Empty inventory namespace is an error": {
			kptfile:     kptFile,
			name:        inventoryName,
			namespace:   "",
			inventoryID: inventoryID,
			force:       false,
			isError:     true,
		},
		"Empty inventory id is an error": {
			kptfile:     kptFile,
			name:        inventoryName,
			namespace:   inventoryNamespace,
			inventoryID: "",
			force:       false,
			isError:     true,
		},
		"Kptfile with inventory already set is error": {
			kptfile:     kptFileWithInventory,
			name:        inventoryName,
			namespace:   inventoryNamespace,
			inventoryID: inventoryID,
			force:       false,
			isError:     true,
		},
		"KptInitOptions default": {
			kptfile:     kptFile,
			name:        inventoryName,
			namespace:   inventoryNamespace,
			inventoryID: inventoryID,
			force:       false,
			isError:     false,
		},
		"KptInitOptions force sets inventory values when already set": {
			kptfile:     kptFileWithInventory,
			name:        inventoryName,
			namespace:   inventoryNamespace,
			inventoryID: inventoryID,
			force:       true,
			isError:     false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Set up fake test factory
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			// Set up temp directory with Ktpfile
			dir, err := ioutil.TempDir("", "kpt-init-options-test")
			assert.NoError(t, err)
			p := filepath.Join(dir, "Kptfile")
			err = ioutil.WriteFile(p, []byte(tc.kptfile), 0600)
			assert.NoError(t, err)

			// Create KptInitOptions and call Run()
			initOptions := NewKptInitOptions(tf, ioStreams)
			initOptions.dir = dir
			initOptions.force = tc.force
			initOptions.name = tc.name
			initOptions.namespace = tc.namespace
			initOptions.inventoryID = tc.inventoryID
			err = initOptions.updateKptfile()

			// Check if there should be an error
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}

			// Otherwise, validate the kptfile values
			assert.NoError(t, err)
			kf, err := kptfileutil.ReadFile(initOptions.dir)
			assert.NoError(t, err)
			assert.Equal(t, inventoryName, kf.Inventory.Name)
			assert.Equal(t, inventoryNamespace, kf.Inventory.Namespace)
			assert.Equal(t, inventoryID, kf.Inventory.InventoryID)
		})
	}
}

func TestInitCmd(t *testing.T) {
	testCases := map[string]struct {
		name       string
		namespace  string
		id         string
		expectedID string
	}{
		"generates an inventory id if one is not provided": {
			name:       "foo",
			namespace:  "bar",
			id:         "",
			expectedID: "0717f9c46d01349e9d575ab1a5131886fb086a43",
		},
		"uses the inventory id if one is provided": {
			name:       "foo",
			namespace:  "bar",
			id:         "abc123",
			expectedID: "abc123",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()
			ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

			dir, err := ioutil.TempDir("", "kpt-init-options-test")
			assert.NoError(t, err)
			kf := kptfile.KptFile{
				ResourceMeta: yaml.ResourceMeta{
					ObjectMeta: yaml.ObjectMeta{
						NameMeta: yaml.NameMeta{
							Name: filepath.Base(dir),
						},
					},
					TypeMeta: yaml.TypeMeta{
						APIVersion: kptfile.TypeMeta.APIVersion,
						Kind:       kptfile.TypeMeta.Kind},
				},
			}
			err = kptfileutil.WriteFile(dir, kf)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			io := NewKptInitOptions(tf, ioStreams)
			io.name = tc.name
			io.namespace = tc.namespace
			io.inventoryID = tc.id
			err = io.Run([]string{dir})
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			newKf, err := kptfileutil.ReadFile(dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			assert.Contains(t, newKf.Inventory.InventoryID, tc.expectedID)
		})
	}
}
