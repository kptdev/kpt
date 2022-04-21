// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var (
	namespaceGVKv1 = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}
)

// InvAddTask encapsulates structures necessary to add/merge inventory
// into the cluster. The InvAddTask should add/merge inventory references
// before the actual object is applied.
type InvAddTask struct {
	TaskName  string
	InvClient inventory.Client
	InvInfo   inventory.Info
	Objects   object.UnstructuredSet
	DryRun    common.DryRunStrategy
}

func (i *InvAddTask) Name() string {
	return i.TaskName
}

func (i *InvAddTask) Action() event.ResourceAction {
	return event.InventoryAction
}

func (i *InvAddTask) Identifiers() object.ObjMetadataSet {
	return object.UnstructuredSetToObjMetadataSet(i.Objects)
}

// Start updates the inventory by merging the locally applied objects
// into the current inventory.
func (i *InvAddTask) Start(taskContext *taskrunner.TaskContext) {
	go func() {
		klog.V(2).Infof("inventory add task starting (name: %q)", i.Name())
		if err := inventory.ValidateNoInventory(i.Objects); err != nil {
			i.sendTaskResult(taskContext, err)
			return
		}
		// Ensures the namespace exists before applying the inventory object into it.
		if invNamespace := inventoryNamespaceInSet(i.InvInfo, i.Objects); invNamespace != nil {
			klog.V(4).Infof("applying inventory namespace %s", invNamespace.GetName())
			if err := i.InvClient.ApplyInventoryNamespace(invNamespace, i.DryRun); err != nil {
				i.sendTaskResult(taskContext, err)
				return
			}
		}
		klog.V(4).Infof("merging %d local objects into inventory", len(i.Objects))
		currentObjs := object.UnstructuredSetToObjMetadataSet(i.Objects)
		_, err := i.InvClient.Merge(i.InvInfo, currentObjs, i.DryRun)
		i.sendTaskResult(taskContext, err)
	}()
}

// Cancel is not supported by the InvAddTask.
func (i *InvAddTask) Cancel(_ *taskrunner.TaskContext) {}

// StatusUpdate is not supported by the InvAddTask.
func (i *InvAddTask) StatusUpdate(_ *taskrunner.TaskContext, _ object.ObjMetadata) {}

// inventoryNamespaceInSet returns the the namespace the passed inventory
// object will be applied to, or nil if this namespace object does not exist
// in the passed slice "infos" or the inventory object is cluster-scoped.
func inventoryNamespaceInSet(inv inventory.Info, objs object.UnstructuredSet) *unstructured.Unstructured {
	if inv == nil {
		return nil
	}
	invNamespace := inv.Namespace()

	for _, obj := range objs {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk == namespaceGVKv1 && obj.GetName() == invNamespace {
			inventory.AddInventoryIDAnnotation(obj, inv)
			return obj
		}
	}
	return nil
}

func (i *InvAddTask) sendTaskResult(taskContext *taskrunner.TaskContext, err error) {
	klog.V(2).Infof("inventory add task completing (name: %q)", i.Name())
	taskContext.TaskChannel() <- taskrunner.TaskResult{
		Err: err,
	}
}
