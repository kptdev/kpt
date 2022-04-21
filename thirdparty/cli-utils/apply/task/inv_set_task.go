// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// InvSetTask encapsulates structures necessary to set the
// inventory references at the end of the apply/prune.
type InvSetTask struct {
	TaskName      string
	InvClient     inventory.Client
	InvInfo       inventory.Info
	PrevInventory object.ObjMetadataSet
	DryRun        common.DryRunStrategy
}

func (i *InvSetTask) Name() string {
	return i.TaskName
}

func (i *InvSetTask) Action() event.ResourceAction {
	return event.InventoryAction
}

func (i *InvSetTask) Identifiers() object.ObjMetadataSet {
	return object.ObjMetadataSet{}
}

// Start sets (creates or replaces) the inventory.
//
// The guiding principal is that anything in the cluster should be in the
// inventory, unless it was explicitly abandoned.
//
// This task must run after all the apply and prune tasks have completed.
//
// Added objects:
// - Applied resources (successful)
//
// Retained objects:
// - Applied resources (filtered/skipped)
// - Applied resources (failed)
// - Deleted resources (filtered/skipped) that were not abandoned
// - Deleted resources (failed)
// - Abandoned resources (failed)
//
// Removed objects:
// - Deleted resources (successful)
// - Abandoned resources (successful)
func (i *InvSetTask) Start(taskContext *taskrunner.TaskContext) {
	go func() {
		klog.V(2).Infof("inventory set task starting (name: %q)", i.Name())
		invObjs := object.ObjMetadataSet{}

		// TODO: Just use InventoryManager.Store()
		im := taskContext.InventoryManager()

		// If an object applied successfully, keep or add it to the inventory.
		appliedObjs := im.SuccessfulApplies()
		klog.V(4).Infof("set inventory %d successful applies", len(appliedObjs))
		invObjs = invObjs.Union(appliedObjs)

		// If an object failed to apply and was previously stored in the inventory,
		// then keep it in the inventory so it can be applied/pruned next time.
		// This will remove new resources that failed to apply from the inventory,
		// because even tho they were added by InvAddTask, the PrevInventory
		// represents the inventory before the pipeline has run.
		applyFailures := i.PrevInventory.Intersection(im.FailedApplies())
		klog.V(4).Infof("keep in inventory %d failed applies", len(applyFailures))
		invObjs = invObjs.Union(applyFailures)

		// If an object skipped apply and was previously stored in the inventory,
		// then keep it in the inventory so it can be applied/pruned next time.
		// It's likely that all the skipped applies are already in the inventory,
		// because the apply filters all currently depend on cluster state,
		// but we're doing the intersection anyway just to be sure.
		applySkips := i.PrevInventory.Intersection(im.SkippedApplies())
		klog.V(4).Infof("keep in inventory %d skipped applies", len(applySkips))
		invObjs = invObjs.Union(applySkips)

		// If an object failed to delete and was previously stored in the inventory,
		// then keep it in the inventory so it can be applied/pruned next time.
		// It's likely that all the delete failures are already in the inventory,
		// because the set of resources to prune comes from the inventory,
		// but we're doing the intersection anyway just to be sure.
		pruneFailures := i.PrevInventory.Intersection(im.FailedDeletes())
		klog.V(4).Infof("set inventory %d failed prunes", len(pruneFailures))
		invObjs = invObjs.Union(pruneFailures)

		// If an object skipped delete and was previously stored in the inventory,
		// then keep it in the inventory so it can be applied/pruned next time.
		// It's likely that all the skipped deletes are already in the inventory,
		// because the set of resources to prune comes from the inventory,
		// but we're doing the intersection anyway just to be sure.
		pruneSkips := i.PrevInventory.Intersection(im.SkippedDeletes())
		klog.V(4).Infof("keep in inventory %d skipped prunes", len(pruneSkips))
		invObjs = invObjs.Union(pruneSkips)

		// If an object is abandoned, then remove it from the inventory.
		abandonedObjects := taskContext.AbandonedObjects()
		klog.V(4).Infof("remove from inventory %d abandoned objects", len(abandonedObjects))
		invObjs = invObjs.Diff(abandonedObjects)

		// If an object is invalid and was previously stored in the inventory,
		// then keep it in the inventory so it can be applied/pruned next time.
		invalidObjects := i.PrevInventory.Intersection(taskContext.InvalidObjects())
		klog.V(4).Infof("keep in inventory %d invalid objects", len(invalidObjects))
		invObjs = invObjs.Union(invalidObjects)

		klog.V(4).Infof("get the apply status for %d objects", len(invObjs))
		objStatus := taskContext.InventoryManager().Inventory().Status.Objects

		klog.V(4).Infof("set inventory %d total objects", len(invObjs))
		err := i.InvClient.Replace(i.InvInfo, invObjs, objStatus, i.DryRun)

		klog.V(2).Infof("inventory set task completing (name: %q)", i.Name())
		taskContext.TaskChannel() <- taskrunner.TaskResult{Err: err}
	}()
}

// Cancel is not supported by the InvSetTask.
func (i *InvSetTask) Cancel(_ *taskrunner.TaskContext) {}

// StatusUpdate is not supported by the InvSetTask.
func (i *InvSetTask) StatusUpdate(_ *taskrunner.TaskContext, _ object.ObjMetadata) {}
