// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

// The solver package is responsible for constructing a
// taskqueue based on the set of resources that should be
// applied.
// This involves setting up the appropriate sequence of
// apply, wait and prune tasks so any dependencies between
// resources doesn't cause a later apply operation to
// fail.
// Currently this package assumes that the resources have
// already been sorted in the appropriate order. We might
// want to consider moving the sorting functionality into
// this package.
package solver

import (
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/apply/task"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/apply/filter"
	"sigs.k8s.io/cli-utils/pkg/apply/info"
	"sigs.k8s.io/cli-utils/pkg/apply/mutator"
	"sigs.k8s.io/cli-utils/pkg/apply/prune"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/object/graph"
	"sigs.k8s.io/cli-utils/pkg/object/validation"
)

type TaskQueueBuilder struct {
	Pruner        *prune.Pruner
	DynamicClient dynamic.Interface
	OpenAPIGetter discovery.OpenAPISchemaInterface
	InfoHelper    info.Helper
	Mapper        meta.RESTMapper
	InvClient     inventory.Client
	// Collector is used to collect validation errors and invalid objects.
	// Invalid objects will be filtered and not be injected into tasks.
	Collector     *validation.Collector
	ApplyFilters  []filter.ValidationFilter
	ApplyMutators []mutator.Interface
	PruneFilters  []filter.ValidationFilter

	// The accumulated tasks and counter variables to name tasks.
	applyCounter int
	pruneCounter int
	waitCounter  int

	invInfo   inventory.Info
	applyObjs object.UnstructuredSet
	pruneObjs object.UnstructuredSet
}

type TaskQueue struct {
	tasks []taskrunner.Task
}

func (tq *TaskQueue) ToChannel() chan taskrunner.Task {
	taskQueue := make(chan taskrunner.Task, len(tq.tasks))
	for _, t := range tq.tasks {
		taskQueue <- t
	}
	return taskQueue
}

func (tq *TaskQueue) ToActionGroups() []event.ActionGroup {
	var ags []event.ActionGroup

	for _, t := range tq.tasks {
		ags = append(ags, event.ActionGroup{
			Name:        t.Name(),
			Action:      t.Action(),
			Identifiers: t.Identifiers(),
		})
	}
	return ags
}

type Options struct {
	ServerSideOptions common.ServerSideOptions
	ReconcileTimeout  time.Duration
	// True if we are destroying, which deletes the inventory object
	// as well (possibly) the inventory namespace.
	Destroy bool
	// True if we're deleting prune objects
	Prune                  bool
	DryRunStrategy         common.DryRunStrategy
	PrunePropagationPolicy metav1.DeletionPropagation
	PruneTimeout           time.Duration
	InventoryPolicy        inventory.Policy
}

// WithInventory sets the inventory info and returns the builder for chaining.
func (t *TaskQueueBuilder) WithInventory(inv inventory.Info) *TaskQueueBuilder {
	t.invInfo = inv
	return t
}

// WithApplyObjects sets the apply objects and returns the builder for chaining.
func (t *TaskQueueBuilder) WithApplyObjects(applyObjs object.UnstructuredSet) *TaskQueueBuilder {
	t.applyObjs = applyObjs
	return t
}

// WithPruneObjects sets the prune objects and returns the builder for chaining.
func (t *TaskQueueBuilder) WithPruneObjects(pruneObjs object.UnstructuredSet) *TaskQueueBuilder {
	t.pruneObjs = pruneObjs
	return t
}

// Build returns the queue of tasks that have been created
func (t *TaskQueueBuilder) Build(taskContext *taskrunner.TaskContext, o Options) *TaskQueue {
	var tasks []taskrunner.Task

	// reset counters
	t.applyCounter = 0
	t.pruneCounter = 0
	t.waitCounter = 0

	// Filter objects that failed earlier validation
	applyObjs := t.Collector.FilterInvalidObjects(t.applyObjs)
	pruneObjs := t.Collector.FilterInvalidObjects(t.pruneObjs)

	// Merge applyObjs & pruneObjs and graph them together.
	// This detects implicit and explicit dependencies.
	// Invalid dependency annotations will be treated as validation errors.
	allObjs := make(object.UnstructuredSet, 0, len(applyObjs)+len(pruneObjs))
	allObjs = append(allObjs, applyObjs...)
	allObjs = append(allObjs, pruneObjs...)
	g, err := graph.DependencyGraph(allObjs)
	if err != nil {
		t.Collector.Collect(err)
	}
	// Store graph for use by DependencyFilter
	taskContext.SetGraph(g)
	// Sort objects into phases (apply order).
	// Cycles will be treated as validation errors.
	idSetList, err := g.Sort()
	if err != nil {
		t.Collector.Collect(err)
	}

	// Filter objects with cycles or invalid dependency annotations
	applyObjs = t.Collector.FilterInvalidObjects(applyObjs)
	pruneObjs = t.Collector.FilterInvalidObjects(pruneObjs)

	if !o.Destroy {
		// InvAddTask creates the inventory and adds any objects being applied
		klog.V(2).Infof("adding inventory add task (%d objects)", len(applyObjs))
		tasks = append(tasks, &task.InvAddTask{
			TaskName:  "inventory-add-0",
			InvClient: t.InvClient,
			InvInfo:   t.invInfo,
			Objects:   applyObjs,
			DryRun:    o.DryRunStrategy,
		})
	}

	if len(applyObjs) > 0 {
		// Register actuation plan in the inventory
		for _, id := range object.UnstructuredSetToObjMetadataSet(applyObjs) {
			taskContext.InventoryManager().AddPendingApply(id)
		}

		// Filter idSetList down to just apply objects
		applySets := graph.HydrateSetList(idSetList, applyObjs)

		for _, applySet := range applySets {
			tasks = append(tasks,
				t.newApplyTask(applySet, t.ApplyFilters, t.ApplyMutators, o))
			// dry-run skips wait tasks
			if !o.DryRunStrategy.ClientOrServerDryRun() {
				applyIds := object.UnstructuredSetToObjMetadataSet(applySet)
				tasks = append(tasks,
					t.newWaitTask(applyIds, taskrunner.AllCurrent, o.ReconcileTimeout))
			}
		}
	}

	if o.Prune && len(pruneObjs) > 0 {
		// Register actuation plan in the inventory
		for _, id := range object.UnstructuredSetToObjMetadataSet(pruneObjs) {
			taskContext.InventoryManager().AddPendingDelete(id)
		}

		// Filter idSetList down to just prune objects
		pruneSets := graph.HydrateSetList(idSetList, pruneObjs)

		// Reverse apply order to get prune order
		graph.ReverseSetList(pruneSets)

		for _, pruneSet := range pruneSets {
			tasks = append(tasks,
				t.newPruneTask(pruneSet, t.PruneFilters, o))
			// dry-run skips wait tasks
			if !o.DryRunStrategy.ClientOrServerDryRun() {
				pruneIds := object.UnstructuredSetToObjMetadataSet(pruneSet)
				tasks = append(tasks,
					t.newWaitTask(pruneIds, taskrunner.AllNotFound, o.PruneTimeout))
			}
		}
	}

	// TODO: add InvSetTask when Destroy=true to retain undeleted objects
	if !o.Destroy {
		klog.V(2).Infoln("adding inventory set task")
		prevInvIds, _ := t.InvClient.GetClusterObjs(t.invInfo)
		tasks = append(tasks, &task.InvSetTask{
			TaskName:      "inventory-set-0",
			InvClient:     t.InvClient,
			InvInfo:       t.invInfo,
			PrevInventory: prevInvIds,
			DryRun:        o.DryRunStrategy,
		})
	} else {
		klog.V(2).Infoln("adding delete inventory task")
		tasks = append(tasks, &task.DeleteInvTask{
			TaskName:  "delete-inventory-0",
			InvClient: t.InvClient,
			InvInfo:   t.invInfo,
			DryRun:    o.DryRunStrategy,
		})
	}

	return &TaskQueue{tasks: tasks}
}

// AppendApplyTask appends a task to the task queue to apply the passed objects
// to the cluster. Returns a pointer to the Builder to chain function calls.
func (t *TaskQueueBuilder) newApplyTask(applyObjs object.UnstructuredSet,
	applyFilters []filter.ValidationFilter, applyMutators []mutator.Interface, o Options) taskrunner.Task {
	applyObjs = t.Collector.FilterInvalidObjects(applyObjs)
	klog.V(2).Infof("adding apply task (%d objects)", len(applyObjs))
	task := &task.ApplyTask{
		TaskName:          fmt.Sprintf("apply-%d", t.applyCounter),
		Objects:           applyObjs,
		Filters:           applyFilters,
		Mutators:          applyMutators,
		ServerSideOptions: o.ServerSideOptions,
		DryRunStrategy:    o.DryRunStrategy,
		DynamicClient:     t.DynamicClient,
		OpenAPIGetter:     t.OpenAPIGetter,
		InfoHelper:        t.InfoHelper,
		Mapper:            t.Mapper,
	}
	t.applyCounter++
	return task
}

// AppendWaitTask appends a task to wait on the passed objects to the task queue.
// Returns a pointer to the Builder to chain function calls.
func (t *TaskQueueBuilder) newWaitTask(waitIds object.ObjMetadataSet, condition taskrunner.Condition,
	waitTimeout time.Duration) taskrunner.Task {
	waitIds = t.Collector.FilterInvalidIds(waitIds)
	klog.V(2).Infoln("adding wait task")
	task := taskrunner.NewWaitTask(
		fmt.Sprintf("wait-%d", t.waitCounter),
		waitIds,
		condition,
		waitTimeout,
		t.Mapper,
	)
	t.waitCounter++
	return task
}

// AppendPruneTask appends a task to delete objects from the cluster to the task queue.
// Returns a pointer to the Builder to chain function calls.
func (t *TaskQueueBuilder) newPruneTask(pruneObjs object.UnstructuredSet,
	pruneFilters []filter.ValidationFilter, o Options) taskrunner.Task {
	pruneObjs = t.Collector.FilterInvalidObjects(pruneObjs)
	klog.V(2).Infof("adding prune task (%d objects)", len(pruneObjs))
	task := &task.PruneTask{
		TaskName:          fmt.Sprintf("prune-%d", t.pruneCounter),
		Objects:           pruneObjs,
		Filters:           pruneFilters,
		Pruner:            t.Pruner,
		PropagationPolicy: o.PrunePropagationPolicy,
		DryRunStrategy:    o.DryRunStrategy,
		Destroy:           o.Destroy,
	}
	t.pruneCounter++
	return task
}
