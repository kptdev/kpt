// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	"sigs.k8s.io/cli-utils/pkg/inventory"
)

var configMapInv = `
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: test-ns
  name: inventory-111111
  labels:
    cli-utils.sigs.k8s.io/inventory-id: XXXX-YYYY-ZZZZ
`

func TestDualDelegatingProvider_Read(t *testing.T) {
	testCases := map[string]struct {
		manifests map[string]string
		numObjs   int
		invKind   string
		isError   bool
	}{
		"Basic ResourceGroup inventory object created": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
				"pod-a.yaml": podA,
			},
			numObjs: 2,
			invKind: "ResourceGroup",
			isError: false,
		},
		"Only ResourceGroup inventory object created": {
			manifests: map[string]string{
				"Kptfile":    kptFile,
			},
			numObjs: 1,
			invKind: "ResourceGroup",
			isError: false,
		},
		"ResourceGroup inventory object with multiple objects": {
			manifests: map[string]string{
				"pod-a.yaml": podA,
				"Kptfile":    kptFile,
				"deployment-a.yaml":       deploymentA,
			},
			numObjs: 3,
			invKind: "ResourceGroup",
			isError: false,
		},
		"Basic ConfigMap inventory object created": {
			manifests: map[string]string{
				"inventory-template.yaml": configMapInv,
				"deployment-a.yaml":       deploymentA,
			},
			numObjs: 2,
			invKind: "ConfigMap",
			isError: false,
		},
		"Only ConfigMap inventory object created": {
			manifests: map[string]string{
				"inventory-template.yaml": configMapInv,
			},
			numObjs: 1,
			invKind: "ConfigMap",
			isError: false,
		},
		"ConfigMap inventory object with multiple objects": {
			manifests: map[string]string{
				"deployment-a.yaml":       deploymentA,
				"inventory-template.yaml": configMapInv,
				"pod-a.yaml": podA,
			},
			numObjs: 3,
			invKind: "ConfigMap",
			isError: false,
		},
		"No inventory manifests is an error": {
			manifests: map[string]string{
				"pod-a.yaml":        podA,
				"deployment-a.yaml": deploymentA,
			},
			numObjs: 2,
			isError: true,
		},
		"Multiple manifests is an error": {
			manifests: map[string]string{
				"inventory-template.yaml": configMapInv,
				"Kptfile":                 kptFile,
				"pod-a.yaml":              podA,
			},
			numObjs: 3,
			isError: true,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			// Create the fake factory
			tf := cmdtesting.NewTestFactory().WithNamespace("test-ns")
			defer tf.Cleanup()
			// Set up the yaml manifests (including Kptfile) in temp dir.
			dir, err := ioutil.TempDir("", "provider-test")
			assert.NoError(t, err)
			for filename, content := range tc.manifests {
				p := filepath.Join(dir, filename)
				err := ioutil.WriteFile(p, []byte(content), 0600)
				assert.NoError(t, err)
			}
			// Create the DualDelegatingProvider
			provider := NewDualDelegatingProvider(tf)
			assert.Equal(t, tf, provider.Factory())
			// Calling InventoryClient before ManifestReader is always an error.
			_, err = provider.InventoryClient()
			if err == nil {
				t.Errorf("expecting error on InventoryClient, but received none.")
			}
			// Read objects using provider ManifestReader.
			mr, err := provider.ManifestReader(nil, []string{dir})
			if tc.isError {
				if err == nil {
					t.Errorf("expected error on ManifestReader, but received none.")
				}
				return
			}
			objs, err := mr.Read()
			assert.NoError(t, err)
			if tc.numObjs != len(objs) {
				t.Errorf("expected to read (%d) objs, got (%d)", tc.numObjs, len(objs))
			}
			// Retrieve single inventory object and validate the kind.
			inv := inventory.FindInventoryObj(objs)
			if inv == nil {
				t.Errorf("inventory object not found")
			}
			if tc.invKind != inv.GetKind() {
				t.Errorf("expected inventory kind (%s), got (%s)", tc.invKind, inv.GetKind())
			}
			// Calling InventoryClient after ManifestReader is valid.
			_, err = provider.InventoryClient()
			if err != nil {
				t.Errorf("unexpected error calling InventoryClient: %s", err)
			}
		})
	}
}
