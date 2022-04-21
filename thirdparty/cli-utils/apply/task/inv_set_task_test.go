// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/apply/cache"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/testutil"
)

var objInvalid = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
	},
}

func TestInvSetTask(t *testing.T) {
	id1 := object.UnstructuredToObjMetadata(obj1)
	id2 := object.UnstructuredToObjMetadata(obj2)
	id3 := object.UnstructuredToObjMetadata(obj3)
	idInvalid := object.UnstructuredToObjMetadata(objInvalid)

	tests := map[string]struct {
		prevInventory  object.ObjMetadataSet
		appliedObjs    object.ObjMetadataSet
		failedApplies  object.ObjMetadataSet
		failedDeletes  object.ObjMetadataSet
		skippedApplies object.ObjMetadataSet
		skippedDeletes object.ObjMetadataSet
		abandonedObjs  object.ObjMetadataSet
		invalidObjs    object.ObjMetadataSet
		expectedObjs   object.ObjMetadataSet
	}{
		"no apply objs, no prune failures; no inventory": {
			expectedObjs: object.ObjMetadataSet{},
		},
		"one apply objs, no prune failures; one inventory": {
			appliedObjs:  object.ObjMetadataSet{id1},
			expectedObjs: object.ObjMetadataSet{id1},
		},
		"no apply objs, one prune failure, in prev inventory; one inventory": {
			prevInventory: object.ObjMetadataSet{id1},
			failedDeletes: object.ObjMetadataSet{id1},
			expectedObjs:  object.ObjMetadataSet{id1},
		},
		"no apply objs, one prune failure, not in prev inventory; no inventory": {
			// aritifical use case: prunes come from the inventory
			failedDeletes: object.ObjMetadataSet{id1},
			expectedObjs:  object.ObjMetadataSet{},
		},
		"one apply objs, one prune failures; one inventory": {
			// aritifical use case: applies and prunes are mutually exclusive.
			// Delete failure overwrites apply success in object status.
			appliedObjs:   object.ObjMetadataSet{id3},
			failedDeletes: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{},
		},
		"two apply objs, two prune failures; three inventory": {
			// aritifical use case: applies and prunes are mutually exclusive
			prevInventory: object.ObjMetadataSet{id2, id3},
			appliedObjs:   object.ObjMetadataSet{id1, id2},
			failedDeletes: object.ObjMetadataSet{id2, id3},
			expectedObjs:  object.ObjMetadataSet{id1, id2, id3},
		},
		"no apply objs, no apply failures, no prune failures; no inventory": {
			failedApplies: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{},
		},
		"one apply failure not in prev inventory; no inventory": {
			failedApplies: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{},
		},
		"one apply obj, one apply failure not in prev inventory; one inventory": {
			appliedObjs:   object.ObjMetadataSet{id2},
			failedApplies: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{id2},
		},
		"one apply obj, one apply failure in prev inventory; one inventory": {
			appliedObjs:   object.ObjMetadataSet{id2},
			failedApplies: object.ObjMetadataSet{id3},
			prevInventory: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{id2, id3},
		},
		"one apply obj, two apply failures with one in prev inventory; two inventory": {
			appliedObjs:   object.ObjMetadataSet{id2},
			failedApplies: object.ObjMetadataSet{id1, id3},
			prevInventory: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{id2, id3},
		},
		"three apply failures with two in prev inventory; two inventory": {
			failedApplies: object.ObjMetadataSet{id1, id2, id3},
			prevInventory: object.ObjMetadataSet{id2, id3},
			expectedObjs:  object.ObjMetadataSet{id2, id3},
		},
		"three apply failures with three in prev inventory; three inventory": {
			failedApplies: object.ObjMetadataSet{id1, id2, id3},
			prevInventory: object.ObjMetadataSet{id2, id3, id1},
			expectedObjs:  object.ObjMetadataSet{id2, id1, id3},
		},
		"one skipped apply from prev inventory; one inventory": {
			prevInventory:  object.ObjMetadataSet{id1},
			skippedApplies: object.ObjMetadataSet{id1},
			expectedObjs:   object.ObjMetadataSet{id1},
		},
		"one skipped apply, no prev inventory; no inventory": {
			skippedApplies: object.ObjMetadataSet{id1},
			expectedObjs:   object.ObjMetadataSet{},
		},
		"one apply obj, one skipped apply, two prev inventory; two inventory": {
			prevInventory:  object.ObjMetadataSet{id1, id2},
			appliedObjs:    object.ObjMetadataSet{id2},
			skippedApplies: object.ObjMetadataSet{id1},
			expectedObjs:   object.ObjMetadataSet{id1, id2},
		},
		"one skipped delete from prev inventory; one inventory": {
			prevInventory:  object.ObjMetadataSet{id1},
			skippedDeletes: object.ObjMetadataSet{id1},
			expectedObjs:   object.ObjMetadataSet{id1},
		},
		"one apply obj, one skipped delete, two prev inventory; two inventory": {
			prevInventory:  object.ObjMetadataSet{id1, id2},
			appliedObjs:    object.ObjMetadataSet{id2},
			skippedDeletes: object.ObjMetadataSet{id1},
			expectedObjs:   object.ObjMetadataSet{id1, id2},
		},
		"two apply obj, one abandoned, three in prev inventory; two inventory": {
			prevInventory: object.ObjMetadataSet{id1, id2, id3},
			appliedObjs:   object.ObjMetadataSet{id1, id2},
			abandonedObjs: object.ObjMetadataSet{id3},
			expectedObjs:  object.ObjMetadataSet{id1, id2},
		},
		"two abandoned, two in prev inventory; no inventory": {
			prevInventory: object.ObjMetadataSet{id2, id3},
			abandonedObjs: object.ObjMetadataSet{id2, id3},
			expectedObjs:  object.ObjMetadataSet{},
		},
		"same obj skipped delete and abandoned, one in prev inventory; no inventory": {
			prevInventory:  object.ObjMetadataSet{id3},
			skippedDeletes: object.ObjMetadataSet{id3},
			abandonedObjs:  object.ObjMetadataSet{id3},
			expectedObjs:   object.ObjMetadataSet{},
		},
		"preserve invalid objects in the inventory": {
			prevInventory: object.ObjMetadataSet{id3, idInvalid},
			appliedObjs:   object.ObjMetadataSet{id3},
			invalidObjs:   object.ObjMetadataSet{idInvalid},
			expectedObjs:  object.ObjMetadataSet{id3, idInvalid},
		},
		"ignore invalid objects not in the inventory": {
			prevInventory: object.ObjMetadataSet{id3},
			appliedObjs:   object.ObjMetadataSet{id3},
			invalidObjs:   object.ObjMetadataSet{idInvalid},
			expectedObjs:  object.ObjMetadataSet{id3},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client := inventory.NewFakeClient(object.ObjMetadataSet{})
			eventChannel := make(chan event.Event)
			resourceCache := cache.NewResourceCacheMap()
			context := taskrunner.NewTaskContext(eventChannel, resourceCache)

			task := InvSetTask{
				TaskName:      taskName,
				InvClient:     client,
				InvInfo:       nil,
				PrevInventory: tc.prevInventory,
			}
			im := context.InventoryManager()
			for _, applyObj := range tc.appliedObjs {
				im.AddSuccessfulApply(applyObj, "unusued-uid", int64(0))
			}
			for _, applyFailure := range tc.failedApplies {
				im.AddFailedApply(applyFailure)
			}
			for _, pruneObj := range tc.failedDeletes {
				im.AddFailedDelete(pruneObj)
			}
			for _, skippedApply := range tc.skippedApplies {
				im.AddSkippedApply(skippedApply)
			}
			for _, skippedDelete := range tc.skippedDeletes {
				im.AddSkippedDelete(skippedDelete)
			}
			for _, abandonedObj := range tc.abandonedObjs {
				context.AddAbandonedObject(abandonedObj)
			}
			for _, invalidObj := range tc.invalidObjs {
				context.AddInvalidObject(invalidObj)
			}
			if taskName != task.Name() {
				t.Errorf("expected task name (%s), got (%s)", taskName, task.Name())
			}
			task.Start(context)
			result := <-context.TaskChannel()
			if result.Err != nil {
				t.Errorf("unexpected error running InvAddTask: %s", result.Err)
			}
			actual, _ := client.GetClusterObjs(nil)
			testutil.AssertEqual(t, tc.expectedObjs, actual,
				"Actual cluster objects (%d) do not match expected cluster objects (%d)",
				len(actual), len(tc.expectedObjs))
		})
	}
}
