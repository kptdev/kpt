// Copyright 2020 Google LLC
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

package live

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var testNamespace = "test-inventory-namespace"
var inventoryObjName = "test-inventory-obj"
var testInventoryLabel = "test-inventory-label"

var inventoryObj = &unstructured.Unstructured{
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

var testDeployment = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-deployment",
	GroupKind: schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	},
}

var testService = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-deployment",
	GroupKind: schema.GroupKind{
		Group: "apps",
		Kind:  "Service",
	},
}

var testPod = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-pod",
	GroupKind: schema.GroupKind{
		Group: "",
		Kind:  "Pod",
	},
}

func TestLoadStore(t *testing.T) {
	tests := map[string]struct {
		inv     *unstructured.Unstructured
		objs    []object.ObjMetadata
		isError bool
	}{
		"Nil inventory is error": {
			inv:     nil,
			objs:    []object.ObjMetadata{},
			isError: true,
		},
		"No inventory objects is valid": {
			inv:     inventoryObj,
			objs:    []object.ObjMetadata{},
			isError: false,
		},
		"Simple test": {
			inv:     inventoryObj,
			objs:    []object.ObjMetadata{testPod},
			isError: false,
		},
		"Test two objects": {
			inv:     inventoryObj,
			objs:    []object.ObjMetadata{testDeployment, testService},
			isError: false,
		},
		"Test three objects": {
			inv:     inventoryObj,
			objs:    []object.ObjMetadata{testDeployment, testService, testPod},
			isError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			wrapped := WrapInventoryObj(tc.inv)
			_ = wrapped.Store(tc.objs)
			invStored, err := wrapped.GetObject()
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			wrapped = WrapInventoryObj(invStored)
			objs, err := wrapped.Load()
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			if !object.SetEquals(tc.objs, objs) {
				t.Fatalf("expected inventory objs (%v), got (%v)", tc.objs, objs)
			}
		})
	}
}

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

func TestIsResourceGroupInventory(t *testing.T) {
	tests := map[string]struct {
		invObj   *unstructured.Unstructured
		expected bool
		isError  bool
	}{
		"Nil inventory is error": {
			invObj:   nil,
			expected: false,
			isError:  true,
		},
		"ConfigMap inventory is false": {
			invObj:   cmInvObj,
			expected: false,
			isError:  false,
		},
		"ResourceGroup inventory is false": {
			invObj:   inventoryObj,
			expected: true,
			isError:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := IsResourceGroupInventory(tc.invObj)
			if tc.isError {
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			if tc.expected != actual {
				t.Errorf("expected inventory as (%t), got (%t)", tc.expected, actual)
			}
		})
	}
}
