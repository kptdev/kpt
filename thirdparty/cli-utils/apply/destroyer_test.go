// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package apply

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/testutil"
)

func TestDestroyerCancel(t *testing.T) {
	testCases := map[string]struct {
		// inventory input to destroyer
		invInfo inventoryInfo
		// objects in the cluster
		clusterObjs object.UnstructuredSet
		// options input to destroyer.Run
		options DestroyerOptions
		// timeout for destroyer.Run
		runTimeout time.Duration
		// timeout for the test
		testTimeout time.Duration
		// fake input events from the status poller
		statusEvents []pollevent.Event
		// expected output status events (async)
		expectedStatusEvents []testutil.ExpEvent
		// expected output events
		expectedEvents []testutil.ExpEvent
		// true if runTimeout is expected to have caused cancellation
		expectRunTimeout bool
	}{
		"cancelled by caller while waiting for deletion": {
			expectRunTimeout: true,
			runTimeout:       2 * time.Second,
			testTimeout:      30 * time.Second,
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "test",
				id:        "test",
				set: object.ObjMetadataSet{
					testutil.ToIdentifier(t, resources["deployment"]),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
			},
			options: DestroyerOptions{
				EmitStatusEvents: true,
				// DeleteTimeout needs to block long enough to cancel the run,
				// otherwise the WaitTask is skipped.
				DeleteTimeout: 1 * time.Minute,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
					},
				},
				// Resource never becomes NotFound, blocking destroyer.Run from exiting
			},
			expectedStatusEvents: []testutil.ExpEvent{
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					// InitTask
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					// PruneTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.DeleteAction,
						GroupName: "prune-0",
						Type:      event.Started,
					},
				},
				{
					// Delete Deployment
					EventType: event.DeleteType,
					DeleteEvent: &testutil.ExpDeleteEvent{
						GroupName:  "prune-0",
						Operation:  event.Deleted,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Error:      nil,
					},
				},
				{
					// PruneTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.DeleteAction,
						GroupName: "prune-0",
						Type:      event.Finished,
					},
				},
				{
					// WaitTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.WaitAction,
						GroupName: "wait-0",
						Type:      event.Started,
					},
				},
				{
					// Deployment reconcile pending.
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				// Deployment never becomes NotFound.
				// WaitTask is expected to be cancelled before DeleteTimeout.
				{
					// WaitTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.WaitAction,
						GroupName: "wait-0",
						Type:      event.Finished, // TODO: add Cancelled event type
					},
				},
				// Inventory cannot be deleted, because the objects still exist,
				// even tho they've been deleted (ex: blocked by finalizer).
				{
					// Error
					EventType: event.ErrorType,
					ErrorEvent: &testutil.ExpErrorEvent{
						Err: context.DeadlineExceeded,
					},
				},
			},
		},
		"completed with timeout": {
			expectRunTimeout: false,
			runTimeout:       10 * time.Second,
			testTimeout:      30 * time.Second,
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "test",
				id:        "test",
				set: object.ObjMetadataSet{
					testutil.ToIdentifier(t, resources["deployment"]),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
			},
			options: DestroyerOptions{
				EmitStatusEvents: true,
				// DeleteTimeout needs to block long enough for completion
				DeleteTimeout: 1 * time.Minute,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.NotFoundStatus,
					},
				},
				// Resource becoming NotFound should unblock destroyer.Run WaitTask
			},
			expectedStatusEvents: []testutil.ExpEvent{
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
					},
				},
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.NotFoundStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					// InitTask
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					// PruneTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.DeleteAction,
						GroupName: "prune-0",
						Type:      event.Started,
					},
				},
				{
					// Delete Deployment
					EventType: event.DeleteType,
					DeleteEvent: &testutil.ExpDeleteEvent{
						GroupName:  "prune-0",
						Operation:  event.Deleted,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Error:      nil,
					},
				},
				{
					// PruneTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.DeleteAction,
						GroupName: "prune-0",
						Type:      event.Finished,
					},
				},
				{
					// WaitTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.WaitAction,
						GroupName: "wait-0",
						Type:      event.Started,
					},
				},
				{
					// Deployment reconcile pending.
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					// Deployment confirmed NotFound.
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					// WaitTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.WaitAction,
						GroupName: "wait-0",
						Type:      event.Finished,
					},
				},
				{
					// DeleteInvTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "delete-inventory-0",
						Type:      event.Started,
					},
				},
				{
					// DeleteInvTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "delete-inventory-0",
						Type:      event.Finished,
					},
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			poller := newFakePoller(tc.statusEvents)

			invInfo := tc.invInfo.toWrapped()

			destroyer := newTestDestroyer(t,
				tc.invInfo,
				// Add the inventory to the cluster (to allow deletion)
				append(tc.clusterObjs, inventory.InvInfoToConfigMap(invInfo)),
				poller,
			)

			// Context for Destroyer.Run
			runCtx, runCancel := context.WithTimeout(context.Background(), tc.runTimeout)
			defer runCancel() // cleanup

			// Context for this test (in case Destroyer.Run never closes the event channel)
			testCtx, testCancel := context.WithTimeout(context.Background(), tc.testTimeout)
			defer testCancel() // cleanup

			eventChannel := destroyer.Run(runCtx, invInfo, tc.options)

			// only start poller once per run
			var once sync.Once
			var events []event.Event

		loop:
			for {
				select {
				case <-testCtx.Done():
					// Test timed out
					runCancel()
					t.Errorf("Destroyer.Run failed to respond to cancellation (expected: %s, timeout: %s)", tc.runTimeout, tc.testTimeout)
					break loop

				case e, ok := <-eventChannel:
					if !ok {
						// Event channel closed
						testCancel()
						break loop
					}
					events = append(events, e)

					if e.Type == event.ActionGroupType &&
						e.ActionGroupEvent.Action == event.WaitAction {
						once.Do(func() {
							// Start sending status events after waiting starts
							poller.Start()
						})
					}
				}
			}

			// Convert events to test events for comparison
			receivedEvents := testutil.EventsToExpEvents(events)

			// Validate & remove expected status events
			for _, e := range tc.expectedStatusEvents {
				var removed int
				receivedEvents, removed = testutil.RemoveEqualEvents(receivedEvents, e)
				if removed < 1 {
					t.Errorf("Expected status event not received: %#v", e)
				}
			}

			// Validate the rest of the events
			testutil.AssertEqual(t, tc.expectedEvents, receivedEvents,
				"Actual events (%d) do not match expected events (%d)",
				len(receivedEvents), len(tc.expectedEvents))

			// Validate that the expected timeout was the cause of the run completion.
			// just in case something else cancelled the run
			if tc.expectRunTimeout {
				assert.Equal(t, context.DeadlineExceeded, runCtx.Err(), "Destroyer.Run exited, but not by expected timeout")
			} else {
				assert.Nil(t, runCtx.Err(), "Destroyer.Run exited, but not by expected timeout")
			}
		})
	}
}
