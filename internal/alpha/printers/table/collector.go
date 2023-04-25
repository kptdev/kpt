// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"fmt"
	"io"
	"sort"
	"sync"

	"k8s.io/klog/v2"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/object/validation"
	"sigs.k8s.io/cli-utils/pkg/print/stats"
	"sigs.k8s.io/cli-utils/pkg/print/table"
)

const InvalidStatus status.Status = "Invalid"

func newResourceStateCollector(resourceGroups []event.ActionGroup, _ io.Writer) *resourceStateCollector {
	resourceInfos := make(map[object.ObjMetadata]*resourceInfo)
	for _, group := range resourceGroups {
		action := group.Action
		// Keep the action that describes the operation for the resource
		// rather than that we will wait for it.
		if action == event.WaitAction {
			continue
		}
		for _, identifier := range group.Identifiers {
			resourceInfos[identifier] = &resourceInfo{
				identifier: identifier,
				resourceStatus: &pe.ResourceStatus{
					Identifier: identifier,
					Status:     status.UnknownStatus,
				},
				ResourceAction: action,
			}
		}
	}
	var ids []object.ObjMetadata
	for id := range resourceInfos {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if ids[i].Namespace != ids[j].Namespace {
			return ids[i].Namespace < ids[j].Namespace
		}
		if ids[i].GroupKind.Group != ids[j].GroupKind.Group {
			return ids[i].GroupKind.Group < ids[j].GroupKind.Group
		}
		if ids[i].GroupKind.Kind != ids[j].GroupKind.Kind {
			return ids[i].GroupKind.Kind < ids[j].GroupKind.Kind
		}
		if ids[i].Name != ids[j].Name {
			return ids[i].Name < ids[j].Name
		}
		return false
	})
	c := &resourceStateCollector{
		resourceInfos: resourceInfos,
	}
	return c
}

// resourceStateCollector consumes the events from the applier
// eventChannel and keeps track of the latest state for all resources.
// It also provides functionality for fetching the latest seen
// state and return it in format that can be used by the
// BaseTablePrinter.
type resourceStateCollector struct {
	mux sync.RWMutex

	// resourceInfos contains a mapping from the unique
	// resource identifier to a ResourceInfo object that captures
	// the latest state for the given resource.
	resourceInfos map[object.ObjMetadata]*resourceInfo

	// stats collect statistics from handled events
	stats stats.Stats

	err error
}

// resourceInfo captures the latest seen state of a single resource.
// This is used for top-level resources that have a ResourceAction
// associated with them.
type resourceInfo struct {
	// identifier contains the information that identifies a
	// single resource.
	identifier object.ObjMetadata

	// resourceStatus contains the latest status information
	// about the resource.
	resourceStatus *pe.ResourceStatus

	// ResourceAction defines the action we are performing
	// on this particular resource. This can be either Apply
	// or Prune.
	ResourceAction event.ResourceAction

	// Error is set if an error occurred trying to perform
	// the desired action on the resource.
	// Error error

	// lastApplyEvent contains the result after
	// a resource has been applied to the cluster.
	lastApplyEvent event.ApplyEvent

	// lastPruneEvent contains the result after
	// a prune operation on a resource
	lastPruneEvent event.PruneEvent

	// lastDeleteEvent contains the result after
	// a delete operation on a resource
	lastDeleteEvent event.DeleteEvent

	// WaitStatus contains the result after
	// a wait operation on a resource
	WaitStatus event.WaitEventStatus
}

// Identifier returns the identifier for the given resource.
func (r *resourceInfo) Identifier() object.ObjMetadata {
	return r.identifier
}

// ResourceStatus returns the latest seen status for the
// resource.
func (r *resourceInfo) ResourceStatus() *pe.ResourceStatus {
	return r.resourceStatus
}

// SubResources returns a slice of Resource which contains
// any resources created and managed by this resource.
func (r *resourceInfo) SubResources() []table.Resource {
	var resources []table.Resource
	for _, res := range r.resourceStatus.GeneratedResources {
		resources = append(resources, &subResourceInfo{
			resourceStatus: res,
		})
	}
	return resources
}

// subResourceInfo captures the latest seen state of a
// single subResource, i.e. resources that are created and
// managed by one of the top-level resources we either apply
// or prune.
type subResourceInfo struct {
	// resourceStatus contains the latest status information
	// about the subResource.
	resourceStatus *pe.ResourceStatus
}

// Identifier returns the identifier for the given subResource.
func (r *subResourceInfo) Identifier() object.ObjMetadata {
	return r.resourceStatus.Identifier
}

// ResourceStatus returns the latest seen status for the
// subResource.
func (r *subResourceInfo) ResourceStatus() *pe.ResourceStatus {
	return r.resourceStatus
}

// SubResources returns a slice of Resource which contains
// any resources created and managed by this resource.
func (r *subResourceInfo) SubResources() []table.Resource {
	var resources []table.Resource
	for _, res := range r.resourceStatus.GeneratedResources {
		resources = append(resources, &subResourceInfo{
			resourceStatus: res,
		})
	}
	return resources
}

// Listen starts a new goroutine that will listen for events on the
// provided eventChannel and keep track of the latest state for
// the resources. The goroutine will exit when the provided
// eventChannel is closed.
// The function returns a channel. When this channel is closed, the
// goroutine has processed all events in the eventChannel and
// exited.
func (r *resourceStateCollector) Listen(eventChannel <-chan event.Event) <-chan listenerResult {
	completed := make(chan listenerResult)
	go func() {
		defer close(completed)
		for ev := range eventChannel {
			if err := r.processEvent(ev); err != nil {
				completed <- listenerResult{err: err}
				return
			}
		}
	}()
	return completed
}

type listenerResult struct {
	err error
}

// processEvent processes an event and updates the state.
func (r *resourceStateCollector) processEvent(ev event.Event) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	switch ev.Type {
	case event.ValidationType:
		return r.processValidationEvent(ev.ValidationEvent)
	case event.StatusType:
		r.processStatusEvent(ev.StatusEvent)
	case event.ApplyType:
		r.processApplyEvent(ev.ApplyEvent)
	case event.PruneType:
		r.processPruneEvent(ev.PruneEvent)
	case event.DeleteType:
		r.processDeleteEvent(ev.DeleteEvent)
	case event.WaitType:
		r.processWaitEvent(ev.WaitEvent)
	case event.ErrorType:
		return ev.ErrorEvent.Err
	}
	return nil
}

// processValidationEvent handles events pertaining to a validation error
// for a resource.
func (r *resourceStateCollector) processValidationEvent(e event.ValidationEvent) error {
	klog.V(7).Infoln("processing validation event")
	// unwrap validation errors
	err := e.Error
	if vErr, ok := err.(*validation.Error); ok {
		err = vErr.Unwrap()
	}
	if len(e.Identifiers) == 0 {
		// no objects, invalid event
		return fmt.Errorf("invalid validation event: no identifiers: %w", err)
	}
	for _, id := range e.Identifiers {
		previous, found := r.resourceInfos[id]
		if !found {
			klog.V(4).Infof("%s status event not found in ResourceInfos; no processing", id)
			continue
		}
		previous.resourceStatus = &pe.ResourceStatus{
			Identifier: id,
			Status:     InvalidStatus,
			Message:    e.Error.Error(),
		}
	}
	return nil
}

// processStatusEvent handles events pertaining to a status
// update for a resource.
func (r *resourceStateCollector) processStatusEvent(e event.StatusEvent) {
	klog.V(7).Infoln("processing status event")
	previous, found := r.resourceInfos[e.Identifier]
	if !found {
		klog.V(4).Infof("%s status event not found in ResourceInfos; no processing", e.Identifier)
		return
	}
	previous.resourceStatus = e.PollResourceInfo
}

// processApplyEvent handles events relating to apply operations
func (r *resourceStateCollector) processApplyEvent(e event.ApplyEvent) {
	identifier := e.Identifier
	klog.V(7).Infof("processing apply event for %s", identifier)
	previous, found := r.resourceInfos[identifier]
	if !found {
		klog.V(4).Infof("%s apply event not found in ResourceInfos; no processing", identifier)
		return
	}
	previous.lastApplyEvent = e

	r.stats.ApplyStats.Inc(e.Status)
}

// processPruneEvent handles event related to prune operations.
func (r *resourceStateCollector) processPruneEvent(e event.PruneEvent) {
	identifier := e.Identifier
	klog.V(7).Infof("processing prune event for %s", identifier)
	previous, found := r.resourceInfos[identifier]
	if !found {
		klog.V(4).Infof("%s prune event not found in ResourceInfos; no processing", identifier)
		return
	}

	previous.lastPruneEvent = e

	r.stats.PruneStats.Inc(e.Status)
}

// processDeleteEvent handles event related to delete operations.
func (r *resourceStateCollector) processDeleteEvent(e event.DeleteEvent) {
	identifier := e.Identifier
	klog.V(7).Infof("processing delete event for %s", identifier)
	previous, found := r.resourceInfos[identifier]
	if !found {
		klog.V(4).Infof("%s delete event not found in ResourceInfos; no processing", identifier)
		return
	}
	previous.lastDeleteEvent = e

	r.stats.DeleteStats.Inc(e.Status)
}

// processPruneEvent handles event related to prune operations.
func (r *resourceStateCollector) processWaitEvent(e event.WaitEvent) {
	identifier := e.Identifier
	klog.V(7).Infof("processing wait event for %s", identifier)
	previous, found := r.resourceInfos[identifier]
	if !found {
		klog.V(4).Infof("%s wait event not found in ResourceInfos; no processing", identifier)
		return
	}
	previous.WaitStatus = e.Status
	r.stats.WaitStats.Inc(e.Status)
}

// ResourceState contains the latest state for all the resources.
type ResourceState struct {
	resourceInfos ResourceInfos

	err error
}

// Resources returns a slice containing the latest state
// for each individual resource.
func (r *ResourceState) Resources() []table.Resource {
	var resources []table.Resource
	for _, res := range r.resourceInfos {
		resources = append(resources, res)
	}
	return resources
}

func (r *ResourceState) Error() error {
	return r.err
}

// LatestState returns a ResourceState object that contains
// a copy of the latest state for all resources.
func (r *resourceStateCollector) LatestState() *ResourceState {
	r.mux.RLock()
	defer r.mux.RUnlock()

	var resourceInfos ResourceInfos
	for _, ri := range r.resourceInfos {
		resourceInfos = append(resourceInfos, &resourceInfo{
			identifier:      ri.identifier,
			resourceStatus:  ri.resourceStatus,
			ResourceAction:  ri.ResourceAction,
			lastApplyEvent:  ri.lastApplyEvent,
			lastDeleteEvent: ri.lastDeleteEvent,
			lastPruneEvent:  ri.lastPruneEvent,
			WaitStatus:      ri.WaitStatus,
		})
	}
	sort.Sort(resourceInfos)

	return &ResourceState{
		resourceInfos: resourceInfos,
		err:           r.err,
	}
}

type ResourceInfos []*resourceInfo

func (g ResourceInfos) Len() int {
	return len(g)
}

func (g ResourceInfos) Less(i, j int) bool {
	idI := g[i].identifier
	idJ := g[j].identifier

	if idI.Namespace != idJ.Namespace {
		return idI.Namespace < idJ.Namespace
	}
	if idI.GroupKind.Group != idJ.GroupKind.Group {
		return idI.GroupKind.Group < idJ.GroupKind.Group
	}
	if idI.GroupKind.Kind != idJ.GroupKind.Kind {
		return idI.GroupKind.Kind < idJ.GroupKind.Kind
	}
	return idI.Name < idJ.Name
}

func (g ResourceInfos) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}
