// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"testing"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var (
	depID = object.ObjMetadata{
		GroupKind: schema.GroupKind{
			Group: "apps",
			Kind:  "Deployment",
		},
		Name:      "Foo",
		Namespace: "Bar",
	}
	customID = object.ObjMetadata{
		GroupKind: schema.GroupKind{
			Group: "custom.io",
			Kind:  "Custom",
		},
		Name: "Custom",
	}
)

const testMessage = "test message for ResourceStatus"

func TestResourceStateCollector_New(t *testing.T) {
	testCases := map[string]struct {
		resourceGroups []event.ActionGroup
		resourceInfos  map[object.ObjMetadata]*ResourceInfo
	}{
		"no resources": {
			resourceGroups: []event.ActionGroup{},
			resourceInfos:  map[object.ObjMetadata]*ResourceInfo{},
		},
		"several resources for apply": {
			resourceGroups: []event.ActionGroup{
				{
					Action: event.ApplyAction,
					Identifiers: []object.ObjMetadata{
						depID, customID,
					},
				},
			},
			resourceInfos: map[object.ObjMetadata]*ResourceInfo{
				depID: {
					ResourceAction: event.ApplyAction,
				},
				customID: {
					ResourceAction: event.ApplyAction,
				},
			},
		},
		"several resources for prune": {
			resourceGroups: []event.ActionGroup{
				{
					Action: event.ApplyAction,
					Identifiers: []object.ObjMetadata{
						customID,
					},
				},
				{
					Action: event.PruneAction,
					Identifiers: []object.ObjMetadata{
						depID,
					},
				},
			},
			resourceInfos: map[object.ObjMetadata]*ResourceInfo{
				depID: {
					ResourceAction: event.PruneAction,
				},
				customID: {
					ResourceAction: event.ApplyAction,
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			rsc := newResourceStateCollector(tc.resourceGroups)

			assert.Equal(t, len(tc.resourceInfos), len(rsc.resourceInfos))
			for expID, expRi := range tc.resourceInfos {
				actRi, found := rsc.resourceInfos[expID]
				if !found {
					t.Errorf("expected to find id %v, but didn't", expID)
				}
				assert.Equal(t, expRi.ResourceAction, actRi.ResourceAction)
			}
		})
	}
}

func TestResourceStateCollector_ProcessStatusEvent(t *testing.T) {
	testCases := map[string]struct {
		resourceGroups []event.ActionGroup
		statusEvent    event.StatusEvent
	}{
		"nil StatusEvent.Resource does not crash": {
			resourceGroups: []event.ActionGroup{},
			statusEvent: event.StatusEvent{
				Resource: nil,
			},
		},
		"unfound Resource identifier does not crash": {
			resourceGroups: []event.ActionGroup{
				{
					Action:      event.ApplyAction,
					Identifiers: []object.ObjMetadata{depID},
				},
			},
			statusEvent: event.StatusEvent{
				PollResourceInfo: &pe.ResourceStatus{
					Identifier: customID, // Does not match identifier in resourceGroups
				},
			},
		},
		"basic status event for applying two resources updates resourceStatus": {
			resourceGroups: []event.ActionGroup{
				{
					Action: event.ApplyAction,
					Identifiers: []object.ObjMetadata{
						depID, customID,
					},
				},
			},
			statusEvent: event.StatusEvent{
				PollResourceInfo: &pe.ResourceStatus{
					Identifier: depID,
					Message:    testMessage,
				},
			},
		},
		"several resources for prune": {
			resourceGroups: []event.ActionGroup{
				{
					Action: event.ApplyAction,
					Identifiers: []object.ObjMetadata{
						customID,
					},
				},
				{
					Action: event.PruneAction,
					Identifiers: []object.ObjMetadata{
						depID,
					},
				},
			},
			statusEvent: event.StatusEvent{
				PollResourceInfo: &pe.ResourceStatus{
					Identifier: depID,
					Message:    testMessage,
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			rsc := newResourceStateCollector(tc.resourceGroups)
			rsc.processStatusEvent(tc.statusEvent)
			id, found := getID(tc.statusEvent)
			if found {
				resourceInfo, found := rsc.resourceInfos[id]
				if found {
					// Validate the ResourceStatus was set from StatusEvent
					if resourceInfo.resourceStatus != tc.statusEvent.PollResourceInfo {
						t.Errorf("status event not processed for %s", id)
					}
				}
			}
		})
	}
}

func getID(e event.StatusEvent) (object.ObjMetadata, bool) {
	if e.Resource == nil {
		return object.ObjMetadata{}, false
	}
	return e.Identifier, true
}
