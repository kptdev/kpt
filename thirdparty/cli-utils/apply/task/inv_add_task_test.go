// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/apply/cache"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var namespace = "test-namespace"

var inventoryObj = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      "test-inventory-obj",
			"namespace": namespace,
			"labels": map[string]interface{}{
				common.InventoryLabel: "test-app-label",
			},
		},
	},
}

var localInv = inventory.WrapInventoryInfoObj(inventoryObj)

var obj1 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":      "obj1",
			"namespace": namespace,
		},
	},
}

var obj2 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      "obj2",
			"namespace": namespace,
		},
	},
}

var obj3 = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      "obj3",
			"namespace": "different-namespace",
		},
	},
}

const taskName = "test-inventory-task"

func TestInvAddTask(t *testing.T) {
	id1 := object.UnstructuredToObjMetadata(obj1)
	id2 := object.UnstructuredToObjMetadata(obj2)
	id3 := object.UnstructuredToObjMetadata(obj3)

	tests := map[string]struct {
		initialObjs  object.ObjMetadataSet
		applyObjs    []*unstructured.Unstructured
		expectedObjs object.ObjMetadataSet
	}{
		"no initial inventory and no apply objects; no merged inventory": {
			initialObjs:  object.ObjMetadataSet{},
			applyObjs:    []*unstructured.Unstructured{},
			expectedObjs: object.ObjMetadataSet{},
		},
		"no initial inventory, one apply object; one merged inventory": {
			initialObjs:  object.ObjMetadataSet{},
			applyObjs:    []*unstructured.Unstructured{obj1},
			expectedObjs: object.ObjMetadataSet{id1},
		},
		"one initial inventory, no apply object; one merged inventory": {
			initialObjs:  object.ObjMetadataSet{id2},
			applyObjs:    []*unstructured.Unstructured{},
			expectedObjs: object.ObjMetadataSet{id2},
		},
		"one initial inventory, one apply object; one merged inventory": {
			initialObjs:  object.ObjMetadataSet{id3},
			applyObjs:    []*unstructured.Unstructured{obj3},
			expectedObjs: object.ObjMetadataSet{id3},
		},
		"three initial inventory, two same objects; three merged inventory": {
			initialObjs:  object.ObjMetadataSet{id1, id2, id3},
			applyObjs:    []*unstructured.Unstructured{obj2, obj3},
			expectedObjs: object.ObjMetadataSet{id1, id2, id3},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := inventory.NewFakeClient(tc.initialObjs)
			eventChannel := make(chan event.Event)
			resourceCache := cache.NewResourceCacheMap()
			context := taskrunner.NewTaskContext(eventChannel, resourceCache)

			task := InvAddTask{
				TaskName:  taskName,
				InvClient: client,
				InvInfo:   nil,
				Objects:   tc.applyObjs,
			}
			if taskName != task.Name() {
				t.Errorf("expected task name (%s), got (%s)", taskName, task.Name())
			}
			applyIds := object.UnstructuredSetToObjMetadataSet(tc.applyObjs)
			if !task.Identifiers().Equal(applyIds) {
				t.Errorf("expected task ids (%s), got (%s)", applyIds, task.Identifiers())
			}
			task.Start(context)
			result := <-context.TaskChannel()
			if result.Err != nil {
				t.Errorf("unexpected error running InvAddTask: %s", result.Err)
			}
			actual, _ := client.GetClusterObjs(nil)
			if !tc.expectedObjs.Equal(actual) {
				t.Errorf("expected merged inventory (%s), got (%s)", tc.expectedObjs, actual)
			}
		})
	}
}

func TestInventoryNamespaceInSet(t *testing.T) {
	inventoryNamespace := createNamespace(namespace)

	tests := map[string]struct {
		inv       inventory.Info
		objects   []*unstructured.Unstructured
		namespace *unstructured.Unstructured
	}{
		"Nil inventory object, no resources returns nil namespace": {
			inv:       nil,
			objects:   []*unstructured.Unstructured{},
			namespace: nil,
		},
		"Inventory object, but no resources returns nil namespace": {
			inv:       localInv,
			objects:   []*unstructured.Unstructured{},
			namespace: nil,
		},
		"Inventory object, resources with no namespace returns nil namespace": {
			inv:       localInv,
			objects:   []*unstructured.Unstructured{obj1, obj2},
			namespace: nil,
		},
		"Inventory object, different namespace returns nil namespace": {
			inv:       localInv,
			objects:   []*unstructured.Unstructured{createNamespace("foo")},
			namespace: nil,
		},
		"Inventory object, inventory namespace returns inventory namespace": {
			inv:       localInv,
			objects:   []*unstructured.Unstructured{obj1, inventoryNamespace, obj3},
			namespace: inventoryNamespace,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actualNamespace := inventoryNamespaceInSet(tc.inv, tc.objects)
			if tc.namespace != actualNamespace {
				t.Fatalf("expected namespace (%v), got (%v)", tc.namespace, actualNamespace)
			}
		})
	}
}

func createNamespace(ns string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": ns,
			},
		},
	}
}
