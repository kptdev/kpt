// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package preprocess

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestPreProcess(t *testing.T) {
	testcases := []struct {
		name            string
		inventoryObject *unstructured.Unstructured
		expected        inventory.InventoryPolicy
	}{
		{
			name:            "nil cluster inventory object",
			inventoryObject: nil,
			expected:        inventory.InventoryPolicyMustMatch,
		},
		{
			name: "existing cluster inventory object without managed-by label",
			inventoryObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test",
						"namespace": "test",
						"labels": map[string]interface{}{
							common.InventoryLabel: "test",
						},
					},
				},
			},
			expected: inventory.AdoptIfNoInventory,
		},
		{
			name: "existing cluster inventory object with managed-by label",
			inventoryObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test",
						"namespace": "test",
						"labels": map[string]interface{}{
							common.InventoryLabel:           "test",
							"apps.kubernetes.io/managed-by": "kpt",
						},
					},
				},
			},
			expected: inventory.InventoryPolicyMustMatch,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			invClient := &fakeInventoryClient{inventory: tc.inventoryObject}
			p := &fakeProvider{invClient: invClient}
			actual, err := PreProcess(p, nil, common.DryRunNone)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if actual != tc.expected {
				t.Fatalf("expected %v but got %v", tc.expected, actual)
			}
		})
	}
}

type fakeProvider struct {
	factory   util.Factory
	invClient inventory.InventoryClient
}

func (p *fakeProvider) Factory() util.Factory {
	return p.factory
}

func (p *fakeProvider) InventoryClient() (inventory.InventoryClient, error) {
	return p.invClient, nil
}

type fakeInventoryClient struct {
	inventory *unstructured.Unstructured
}

func (f *fakeInventoryClient) GetClusterObjs(inv inventory.InventoryInfo) ([]object.ObjMetadata, error) {
	return nil, nil
}

func (f *fakeInventoryClient) Merge(inv inventory.InventoryInfo, objs []object.ObjMetadata) ([]object.ObjMetadata, error) {
	return nil, nil
}

// Replace replaces the set of objects stored in the inventory
// object with the passed set of objects, or an error if one occurs.
func (f *fakeInventoryClient) Replace(inv inventory.InventoryInfo, objs []object.ObjMetadata) error {
	return nil
}

// DeleteInventoryObj deletes the passed inventory object from the APIServer.
func (f *fakeInventoryClient) DeleteInventoryObj(inv inventory.InventoryInfo) error {
	return nil
}

// SetDryRunStrategy sets the dry run strategy on whether this we actually mutate.
func (f *fakeInventoryClient) SetDryRunStrategy(drs common.DryRunStrategy) {}

// ApplyInventoryNamespace applies the Namespace that the inventory object should be in.
func (f *fakeInventoryClient) ApplyInventoryNamespace(invNamespace *unstructured.Unstructured) error {
	return nil
}

// GetClusterInventoryInfo returns the cluster inventory object.
func (f *fakeInventoryClient) GetClusterInventoryInfo(inv inventory.InventoryInfo) (*unstructured.Unstructured, error) {
	return f.inventory, nil
}

// UpdateLabels updates the labels of the cluster inventory object if it exists.
func (f *fakeInventoryClient) UpdateLabels(inv inventory.InventoryInfo, labels map[string]string) error {
	f.inventory.SetLabels(labels)
	return nil
}
