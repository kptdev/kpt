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
	"context"
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/apis/actuation"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	testutil "sigs.k8s.io/cli-utils/pkg/testutil"
)

var testNamespace = "test-inventory-namespace"
var inventoryObjName = "test-inventory-obj"
var testInventoryLabel = "test-inventory-label"

func inventoryWithObjs(objs object.ObjMetadataSet) *unstructured.Unstructured {
	var allObjs []interface{}
	for _, obj := range objs {
		allObjs = append(allObjs, idToUnstructuredMap(obj))
	}

	u := &unstructured.Unstructured{
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
				"resources": allObjs,
			},
		},
	}

	return u
}

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
		inv       *unstructured.Unstructured
		objs      []object.ObjMetadata
		objStatus []actuation.ObjectStatus
		isError   bool
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
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testPod},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testPod),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationPending,
					Reconcile:       actuation.ReconcilePending,
				},
			},
			isError: false,
		},
		"Test two objects": {
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testDeployment, testService},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testDeployment),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testService),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
			},
			isError: false,
		},
		"Test three objects": {
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testDeployment, testService, testPod},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testDeployment),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testService),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testPod),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationPending,
					Reconcile:       actuation.ReconcilePending,
				},
			},
			isError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			wrapped := WrapInventoryObj(0)(tc.inv)
			_ = wrapped.Store(tc.objs, tc.objStatus)
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
			wrapped = WrapInventoryObj(0)(invStored)
			objs, err := wrapped.Load()
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			if !objs.Equal(tc.objs) {
				t.Fatalf("expected inventory objs (%v), got (%v)", tc.objs, objs)
			}
			resourceStatus, _, err := unstructured.NestedSlice(invStored.Object, "status", "resourceStatuses")
			if err != nil {
				t.Fatalf("unexpected error %v received", err)
			}
			if len(resourceStatus) != len(tc.objStatus) {
				t.Fatalf("expected %d resource status but got %d", len(tc.objStatus), len(resourceStatus))
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

func TestMin(t *testing.T) {
	tests := []struct {
		name       string
		i, j, want int
	}{
		{
			"Happy Path",
			1, 2, 1,
		},
		{
			"Happy Path Flipped",
			2, 1, 1,
		},
		{
			"Should not error on same value",
			99, 99, 99,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := min(tc.i, tc.j); got != tc.want {
				t.Errorf("min() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	testCases := map[string]struct {
		objMetas           []object.ObjMetadata
		maxObjectsPerShard int
		existingRG         *unstructured.Unstructured
		expNumberRGObjects int
	}{
		"Add no new objects should create 1 new resourcegroup": {
			objMetas:           nil,
			existingRG:         nil,
			expNumberRGObjects: 1,
		},
		"Add no new objects should create 1 new resourcegroup with sharding enabled": {
			objMetas:           nil,
			maxObjectsPerShard: 5,
			existingRG:         nil,
			expNumberRGObjects: 1,
		},
		"Add add 5 new objects no sharding": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			},
			existingRG:         nil,
			expNumberRGObjects: 1,
		},
		"Add add 5 new objects with max 1 GKNN per RG should create 5 ResourceGroups": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			},
			maxObjectsPerShard: 1,
			existingRG:         nil,
			expNumberRGObjects: 5,
		},
		"Add add 5 new objects with max 3 GKNN per RG should create 2 ResourceGroups": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			},
			maxObjectsPerShard: 3,
			existingRG:         nil,
			expNumberRGObjects: 2,
		},
		"Add 2 new objects to existing ResourceGroup with no sharding": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			},
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 1,
		},
		"Add 2 new objects to existing ResourceGroup with 3 GKNN per RG sharding should have 2 on-cluster RG objects": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			},
			maxObjectsPerShard: 3,
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 2,
		},
		"Add add 0 new objects to existing ResourceGroup with no sharding should add sharded label": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			},
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 1,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(T *testing.T) {
			applyRunner(t, tn, tc, "Apply")
		})
	}
}

func TestApplyWithPrune(t *testing.T) {
	testCases := map[string]applyTestCase{
		"Remove 3 objects from 5 no sharding": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
			},
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 1,
		},
		"Remove 2 objects from 5 with 1 GKNN/RG sharding": {
			objMetas: []object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
			},
			maxObjectsPerShard: 1,
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 3,
		},
		"Remove all objects from 5 should return 1 empty RG": {
			objMetas: nil,
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 1,
		},
		"Remove all objects from 5 with sharding should return 1 empty RG": {
			objMetas:           nil,
			maxObjectsPerShard: 1,
			existingRG: inventoryWithObjs([]object.ObjMetadata{
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-1",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-2",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-3",
					Namespace: "foo",
				},
				{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Name:      "test-deployment-4",
					Namespace: "foo",
				},
			}),
			expNumberRGObjects: 1,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(T *testing.T) {
			applyRunner(t, tn, tc, "ApplyWithPrune")
		})
	}
}

type applyTestCase struct {
	objMetas           []object.ObjMetadata
	maxObjectsPerShard int
	existingRG         *unstructured.Unstructured
	expNumberRGObjects int
}

func applyRunner(t *testing.T, tn string, tc applyTestCase, runType string) {
	dc, mapper := fakeClient()
	irg := InventoryResourceGroup{
		inv:           inventoryObj,
		objMetas:      tc.objMetas,
		strategy:      inventory.NameStrategy,
		resourceCount: tc.maxObjectsPerShard,
	}
	namespacedClient, err := irg.getNamespacedClient(dc, mapper)
	if err != nil {
		t.Error(err)
	}

	if tc.existingRG != nil {
		_, err := namespacedClient.Create(context.TODO(), tc.existingRG, v1.CreateOptions{})
		if err != nil {
			t.Errorf("unable to create existing RG: %v", err)
		}
	}

	switch runType {
	case "ApplyWithPrune":
		err = irg.ApplyWithPrune(dc, mapper, inventory.StatusPolicyAll, tc.objMetas)
	case "Apply":
		err = irg.Apply(dc, mapper, inventory.StatusPolicyAll)
	}
	if err != nil {
		t.Error(err)
	}

	compareApplyResults(t, tc, tn, namespacedClient, irg)

	// Re-run apply functions to ensure it results in a no-op the second time around.
	switch runType {
	case "ApplyWithPrune":
		err = irg.ApplyWithPrune(dc, mapper, inventory.StatusPolicyAll, tc.objMetas)
	case "Apply":
		err = irg.Apply(dc, mapper, inventory.StatusPolicyAll)
	}
	if err != nil {
		t.Error(err)
	}

	compareApplyResults(t, tc, tn+": Run 2", namespacedClient, irg)
}

// compareApplyResults ensures that we get the expected objMetadata written to
// the ResourceGroups.
func compareApplyResults(t *testing.T, tc applyTestCase, tn string,
	namespacedClient dynamic.ResourceInterface, irg InventoryResourceGroup) {
	clusterRGList, err := namespacedClient.List(context.TODO(), v1.ListOptions{
		LabelSelector: irg.shardedLabel(),
	})
	if err != nil {
		t.Error(err)
	}

	// Check we applied the ResourceGroups correctly.
	resourcegroups := clusterRGList.Items

	if len(resourcegroups) != tc.expNumberRGObjects {
		t.Errorf("expected %d ResourceGroup objects, but found %d %+v", tc.expNumberRGObjects, len(resourcegroups), clusterRGList)
	}

	var allResources []object.ObjMetadata
	for _, rg := range resourcegroups {
		objs, err := irg.ReadResourceGroupObjects(&rg)
		if err != nil {
			t.Error(err)
		}
		allResources = append(allResources, objs...)
	}

	sortObjMetas(tc.objMetas)
	sortObjMetas(allResources)

	testutil.AssertEqual(t, tc.objMetas, allResources, tn)
}

func sortObjMetas(objMetas []object.ObjMetadata) {
	sort.Slice(objMetas, func(i, j int) bool {
		return objMetas[i].Name < objMetas[j].Name
	})
}

func fakeClient() (*dynamicfake.FakeDynamicClient, meta.RESTMapper) {
	mapper := testutil.NewFakeRESTMapper(
		appsv1.SchemeGroupVersion.WithKind("Deployment"),
		ResourceGroupGVK,
	)

	dc := dynamicfake.
		NewSimpleDynamicClientWithCustomListKinds(scheme.Scheme,
			map[schema.GroupVersionResource]string{{Group: "kpt.dev", Version: "v1alpha1", Resource: "resourcegroups"}: "ResourceGroupList"})

	return dc, mapper
}
