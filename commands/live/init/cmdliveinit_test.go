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

package init

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/internal/testutil"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	rgfilev1alpha1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1alpha1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// resourceGroupInventoryWithID is a fully-initialized ResourceGroup with an inventory-id label.
var resourceGroupInventoryWithID = `
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: foo
  namespace: test-namespace
  labels:
    cli-utils.sigs.k8s.io/inventory-id: SSSSSSSSSS-RRRRR
`

func TestValidateName(t *testing.T) {
	testCases := map[string]struct {
		name        string
		expectError bool
		errContains string
	}{
		"valid lowercase name": {
			name: "my-app-staging",
		},
		"dots are accepted in subdomains": {
			name: "my.app.v1",
		},
		"empty string is rejected": {
			name:        "",
			expectError: true,
			errContains: "--name is required",
		},
		"whitespace-only is rejected": {
			name:        "   ",
			expectError: true,
			errContains: "--name is required",
		},
		"uppercase is rejected": {
			name:        "MyApp",
			expectError: true,
			errContains: "not a valid Kubernetes resource name",
		},
		"underscore is rejected": {
			name:        "my_app",
			expectError: true,
			errContains: "not a valid Kubernetes resource name",
		},
		"special chars are rejected": {
			name:        "my-app!",
			expectError: true,
			errContains: "not a valid Kubernetes resource name",
		},
		"starts with dash is rejected": {
			name:        "-my-app",
			expectError: true,
			errContains: "not a valid Kubernetes resource name",
		},
		"exceeds 253 chars is rejected": {
			name:        strings.Repeat("a", 254),
			expectError: true,
			errContains: "not a valid Kubernetes resource name",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result, err := validateName(tc.name)
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, strings.TrimSpace(tc.name), result)
			}
		})
	}
}

func TestCmd_generateHash(t *testing.T) {
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
		"Namespace/name hash is deterministic": {
			name:      inventoryName,
			namespace: inventoryNamespace,
			expected:  "b71156e872dad0b8efe1ce0303da20ef583453d6",
			isError:   false,
		},
		"Same inputs produce same hash": {
			name:      inventoryName,
			namespace: inventoryNamespace,
			expected:  "b71156e872dad0b8efe1ce0303da20ef583453d6",
			isError:   false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			actual, err := generateHash(tc.namespace, tc.name)
			if tc.isError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, actual, 40, "SHA-1 hex must be 40 chars (valid label value)")
			assert.Equal(t, tc.expected, actual)
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
		expectedInventory kptfilev1.Inventory
	}{
		"Missing name is an error": {
			kptfile:          kptFile,
			name:             "",
			rgfilename:       rgfilev1alpha1.RGFileName,
			namespace:        "testns",
			inventoryID:      "",
			expectedErrorMsg: "--name is required",
		},
		"Whitespace-only name is rejected": {
			kptfile:          kptFile,
			name:             "   ",
			rgfilename:       rgfilev1alpha1.RGFileName,
			namespace:        "testns",
			inventoryID:      "",
			expectedErrorMsg: "--name is required",
		},
		"Invalid DNS name is rejected": {
			kptfile:          kptFile,
			name:             "My_App!",
			rgfilename:       rgfilev1alpha1.RGFileName,
			namespace:        "testns",
			inventoryID:      "",
			expectedErrorMsg: "not a valid Kubernetes resource name",
		},
		"Explicit inventory-id is preserved when both flags are set": {
			kptfile:     kptFile,
			rgfilename:  rgfilev1alpha1.RGFileName,
			name:        "my-pkg",
			namespace:   "my-ns",
			inventoryID: "custom-legacy-id-123",
			expectedInventory: kptfilev1.Inventory{
				Namespace:   "my-ns",
				Name:        "my-pkg",
				InventoryID: "custom-legacy-id-123",
			},
		},
		"Provided name derives deterministic inventory ID": {
			kptfile:     kptFile,
			rgfilename:  rgfilev1alpha1.RGFileName,
			name:        "my-pkg",
			namespace:   "my-ns",
			inventoryID: "",
			expectedInventory: kptfilev1.Inventory{
				Namespace: "my-ns",
				Name:      "my-pkg",
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
		"Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			name:             inventoryName,
			rgfilename:       "custom-rg.yaml",
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"ResourceGroup without inventory-id is legacy error": {
			kptfile:          kptFile,
			resourcegroup:    resourceGroupInventory,
			rgfilename:       rgfilev1alpha1.RGFileName,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "found existing ResourceGroup without an inventory-id label",
		},
		"ResourceGroup with inventory already set is error": {
			kptfile:          kptFile,
			resourcegroup:    resourceGroupInventoryWithID,
			rgfilename:       rgfilev1alpha1.RGFileName,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set for package",
		},
		"ResourceGroup with inventory and Kptfile with inventory already set is error": {
			kptfile:          kptFileWithInventory,
			resourcegroup:    resourceGroupInventoryWithID,
			rgfilename:       rgfilev1alpha1.RGFileName,
			name:             inventoryName,
			namespace:        inventoryNamespace,
			inventoryID:      inventoryID,
			force:            false,
			expectedErrorMsg: "inventory information already set",
		},
		"The force flag allows changing inventory information even if already set in Kptfile": {
			kptfile:     kptFileWithInventory,
			name:        inventoryName,
			rgfilename:  rgfilev1alpha1.RGFileName,
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
			resourcegroup: resourceGroupInventoryWithID,
			rgfilename:    rgfilev1alpha1.RGFileName,
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
			assert.Equal(t, expectedInv.Name, actualInv.Name)
			assert.Equal(t, expectedInv.Namespace, actualInv.Namespace)
			if expectedInv.InventoryID != "" {
				assert.Equal(t, expectedInv.InventoryID, actualInv.InventoryID)
			} else {
				// Verify deterministic derivation: same name+namespace always yields same hash.
				expectedHash, err := generateHash(actualInv.Namespace, actualInv.Name)
				assert.NoError(t, err)
				assert.Equal(t, expectedHash, actualInv.InventoryID)
			}
		})
	}
}

func TestGenerateHash_DifferentInputs(t *testing.T) {
	testCases := []struct {
		desc     string
		ns       string
		name     string
		expected string
	}{
		{"short pair ab:cd", "ab", "cd", "6d6a43180d720d2526a9c90829cde33f9b36dbdb"},
		{"my-ns:my-pkg", "my-ns", "my-pkg", "6ebf2b6944e9fc957759dd2405ff3879d06197f7"},
		{"ns-a:name-a", "ns-a", "name-a", "01a2429bd398b1d880a145b9f6a40c091119ca7a"},
		{"ns-a:name-b", "ns-a", "name-b", "a7a0df7b43e5aafeb3161a480aa7e68fdc8f3201"},
		{"ns-b:name-a", "ns-b", "name-a", "5ba3e66f5bcd5729b91904f0a7fcc78141644db6"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := generateHash(tc.ns, tc.name)
			require.NoError(t, err)
			assert.Len(t, got, 40)
			assert.Equal(t, tc.expected, got)
		})
	}

	// Changing either namespace or name must change the hash.
	h1, _ := generateHash("ns-a", "name-a")
	h2, _ := generateHash("ns-a", "name-b")
	h3, _ := generateHash("ns-b", "name-a")
	assert.NotEqual(t, h1, h2, "different name should produce different hash")
	assert.NotEqual(t, h1, h3, "different namespace should produce different hash")
	assert.NotEqual(t, h2, h3, "both differ should produce different hash")
}

func TestGenerateHash_NoSeparatorAmbiguity(t *testing.T) {
	// These inputs would collide without length-prefixed encoding:
	//   "a" + "bcd"  vs  "abc" + "d"
	// With the format "%d:%s:%d:%s":
	//   "1:a:3:bcd"  vs  "3:abc:1:d"
	h1, err := generateHash("a", "bcd")
	require.NoError(t, err)
	h2, err := generateHash("abc", "d")
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2, "length-prefixed encoding must prevent separator ambiguity")

	// Also verify the exact expected values.
	assert.Equal(t, "a1724ac2a61ec038d055881eb4403c74ab4256e9", h1)
	assert.Equal(t, "f99cca29ebcfd3bca8c3605d253e4fec27b917ae", h2)
}

func TestConfigureInventoryInfo_InvalidDirectoryNameFallback(t *testing.T) {
	// Internal callers (e.g. migrate) pass Name="" with InventoryID set.
	// ConfigureInventoryInfo.Run() derives the name from the package directory.
	// If the directory name fails IsDNS1123Subdomain, Run must return an error.
	tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
	defer tf.Cleanup()

	parentDir, err := os.MkdirTemp("", "kpt-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(parentDir)

	invalidDir := filepath.Join(parentDir, "UPPER_invalid")
	require.NoError(t, os.MkdirAll(invalidDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(invalidDir, kptfilev1.KptFileName),
		[]byte(kptFile), 0600))

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, invalidDir)
	require.NoError(t, err)

	c := &ConfigureInventoryInfo{
		Pkg:         p,
		Factory:     tf,
		Quiet:       true,
		Name:        "",
		InventoryID: "explicit-inv-id-123",
		RGFileName:  rgfilev1alpha1.RGFileName,
	}
	err = c.Run(fake.CtxWithDefaultPrinter())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid Kubernetes resource name")
	assert.Contains(t, err.Error(), "--name was not provided")
}

func TestConfigureInventoryInfo_ValidDirectoryNameFallback(t *testing.T) {
	// Happy path for internal callers: Name="" with InventoryID set,
	// and the directory name IS a valid DNS-1123 subdomain.
	tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
	defer tf.Cleanup()

	parentDir, err := os.MkdirTemp("", "kpt-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(parentDir)

	validDir := filepath.Join(parentDir, "my-valid-pkg")
	require.NoError(t, os.MkdirAll(validDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(validDir, kptfilev1.KptFileName),
		[]byte(kptFile), 0600))

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, validDir)
	require.NoError(t, err)

	c := &ConfigureInventoryInfo{
		Pkg:         p,
		Factory:     tf,
		Quiet:       true,
		Name:        "",
		InventoryID: "explicit-inv-id-456",
		RGFileName:  rgfilev1alpha1.RGFileName,
	}
	err = c.Run(fake.CtxWithDefaultPrinter())

	require.NoError(t, err)

	rg, err := pkg.ReadRGFile(validDir, rgfilev1alpha1.RGFileName)
	require.NoError(t, err)
	require.NotNil(t, rg)
	assert.Equal(t, "my-valid-pkg", rg.Name)
	assert.Equal(t, "explicit-inv-id-456", rg.Labels[rgfilev1alpha1.RGInventoryIDLabel])
}

func TestConfigureInventoryInfo_DottedDirectoryNameFallback(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
	defer tf.Cleanup()

	parentDir, err := os.MkdirTemp("", "kpt-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(parentDir)

	validDir := filepath.Join(parentDir, "my.valid.pkg")
	require.NoError(t, os.MkdirAll(validDir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(validDir, kptfilev1.KptFileName),
		[]byte(kptFile), 0600))

	p, err := pkg.New(filesys.FileSystemOrOnDisk{}, validDir)
	require.NoError(t, err)

	c := &ConfigureInventoryInfo{
		Pkg:         p,
		Factory:     tf,
		Quiet:       true,
		Name:        "",
		InventoryID: "explicit-inv-id-789",
		RGFileName:  rgfilev1alpha1.RGFileName,
	}
	err = c.Run(fake.CtxWithDefaultPrinter())

	require.NoError(t, err)

	rg, err := pkg.ReadRGFile(validDir, rgfilev1alpha1.RGFileName)
	require.NoError(t, err)
	require.NotNil(t, rg)
	assert.Equal(t, "my.valid.pkg", rg.Name)
	assert.Equal(t, "explicit-inv-id-789", rg.Labels[rgfilev1alpha1.RGInventoryIDLabel])
}

func TestCmd_MissingNameFlagReturnsError(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	w, clean := testutil.SetupWorkspace(t)
	defer clean()
	err := os.WriteFile(filepath.Join(w.WorkspaceDirectory, "Kptfile"),
		[]byte(kptFile), 0600)
	require.NoError(t, err)

	revert := testutil.Chdir(t, w.WorkspaceDirectory)
	defer revert()

	runner := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner.RGFileName = rgfilev1alpha1.RGFileName
	runner.Command.SetArgs([]string{})

	err = runner.Command.Execute()
	require.Error(t, err)
	// Cobra enforces --name as required and returns its own error message.
	assert.Contains(t, err.Error(), "\"name\"")
}

// TestRoundTrip_SameNameProducesSameInventoryID is the key test for issue #4387:
// if a user loses their local package and re-initializes with the same --name
// and same namespace, the inventory-id MUST be identical, so kpt live apply
// can reconnect to existing cluster resources.
func TestRoundTrip_SameNameProducesSameInventoryID(t *testing.T) {
	tf := cmdtesting.NewTestFactory().WithNamespace("my-namespace")
	defer tf.Cleanup()
	ioStreams, _, _, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled

	const deploymentName = "my-app-staging"

	// --- First initialization ---
	w1, clean1 := testutil.SetupWorkspace(t)
	defer clean1()
	require.NoError(t, os.WriteFile(
		filepath.Join(w1.WorkspaceDirectory, kptfilev1.KptFileName),
		[]byte(kptFile), 0600))

	revert1 := testutil.Chdir(t, w1.WorkspaceDirectory)
	defer revert1()

	runner1 := NewRunner(fake.CtxWithDefaultPrinter(), tf, ioStreams)
	runner1.RGFileName = rgfilev1alpha1.RGFileName
	runner1.Command.SetArgs([]string{
		"--name", deploymentName,
	})
	require.NoError(t, runner1.Command.Execute())

	rg1, err := pkg.ReadRGFile(w1.WorkspaceDirectory, rgfilev1alpha1.RGFileName)
	require.NoError(t, err)
	require.NotNil(t, rg1)
	firstInvID := rg1.Labels[rgfilev1alpha1.RGInventoryIDLabel]
	assert.NotEmpty(t, firstInvID, "first init must produce an inventory-id")

	// --- Simulate losing the package: create a brand-new workspace ---
	revert1() // leave the first directory

	w2, clean2 := testutil.SetupWorkspace(t)
	defer clean2()
	require.NoError(t, os.WriteFile(
		filepath.Join(w2.WorkspaceDirectory, kptfilev1.KptFileName),
		[]byte(kptFile), 0600))

	revert2 := testutil.Chdir(t, w2.WorkspaceDirectory)
	defer revert2()

	// Re-create factory with same namespace to ensure same context.
	tf2 := cmdtesting.NewTestFactory().WithNamespace("my-namespace")
	defer tf2.Cleanup()

	runner2 := NewRunner(fake.CtxWithDefaultPrinter(), tf2, ioStreams)
	runner2.RGFileName = rgfilev1alpha1.RGFileName
	runner2.Command.SetArgs([]string{
		"--name", deploymentName,
	})
	require.NoError(t, runner2.Command.Execute())

	rg2, err := pkg.ReadRGFile(w2.WorkspaceDirectory, rgfilev1alpha1.RGFileName)
	require.NoError(t, err)
	require.NotNil(t, rg2)
	secondInvID := rg2.Labels[rgfilev1alpha1.RGInventoryIDLabel]

	// --- THE KEY ASSERTION ---
	// Same name + same namespace MUST produce the exact same inventory-id,
	// so kpt live apply can reconnect to the existing deployment.
	assert.Equal(t, firstInvID, secondInvID,
		"re-initializing with the same --name and namespace must produce the same inventory-id (issue #4387)")

	// Also verify the name and namespace are correct.
	assert.Equal(t, deploymentName, rg1.Name)
	assert.Equal(t, deploymentName, rg2.Name)
	assert.Equal(t, "my-namespace", rg1.Namespace)
	assert.Equal(t, "my-namespace", rg2.Namespace)
}
