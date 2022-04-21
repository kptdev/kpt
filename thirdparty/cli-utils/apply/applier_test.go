// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package apply

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/multierror"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/object/validation"
	"sigs.k8s.io/cli-utils/pkg/testutil"
)

var (
	codec     = scheme.Codecs.LegacyCodec(scheme.Scheme.PrioritizedVersionsAllGroups()...)
	resources = map[string]string{
		"deployment": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  namespace: default
  uid: dep-uid
  generation: 1
spec:
  replicas: 1
`,
		"secret": `
apiVersion: v1
kind: Secret
metadata:
  name: secret
  namespace: default
  uid: secret-uid
  generation: 1
type: Opaque
spec:
  foo: bar
`,
		"inventory": `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-inventory-obj
  namespace: test-namespace
  labels:
    cli-utils.sigs.k8s.io/inventory-id: test-app-label
data: {}
`,
		"obj1": `
apiVersion: v1
kind: Pod
metadata:
  name: obj1
  namespace: test-namespace
spec: {}
`,
		"obj2": `
apiVersion: v1
kind: Pod
metadata:
  name: obj2
  namespace: test-namespace
spec: {}
`,
		"clusterScopedObj": `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-scoped-1
`,
	}
)

//nolint:dupl // event lists are very similar
func TestApplier(t *testing.T) {
	testCases := map[string]struct {
		namespace string
		// resources input to applier
		resources object.UnstructuredSet
		// inventory input to applier
		invInfo inventoryInfo
		// objects in the cluster
		clusterObjs object.UnstructuredSet
		// options input to applier.Run
		options ApplierOptions
		// fake input events from the status poller
		statusEvents []pollevent.Event
		// expected output status events (async)
		expectedStatusEvents []testutil.ExpEvent
		// expected output events
		expectedEvents []testutil.ExpEvent
		// true if runTimeout is expected to have caused cancellation
		expectRunTimeout bool
		// true if testTimeout is expected to have caused cancellation
		expectTestTimeout bool
	}{
		"initial apply without status or prune": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
			},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "default",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				NoPrune:         true,
				InventoryPolicy: inventory.PolicyMustMatch,
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},

				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created, // Create new
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				// Timeout waiting for status event saying deployment is current
				// TODO: update inventory after timeout
				// {
				// 	EventType: event.ActionGroupType,
				// 	ActionGroupEvent: &testutil.ExpActionGroupEvent{
				// 		GroupName: "wait-0",
				// 		Action:    event.WaitAction,
				// 		Type:      event.Finished,
				// 	},
				// },
				// {
				// 	EventType: event.ActionGroupType,
				// 	ActionGroupEvent: &testutil.ExpActionGroupEvent{
				// 		GroupName: "inventory-set-0",
				// 		Action:    event.InventoryAction,
				// 		Type:      event.Started,
				// 	},
				// },
				// {
				// 	EventType: event.ActionGroupType,
				// 	ActionGroupEvent: &testutil.ExpActionGroupEvent{
				// 		GroupName: "inventory-set-0",
				// 		Action:    event.InventoryAction,
				// 		Type:      event.Finished,
				// 	},
				// },
			},
			expectTestTimeout: true,
		},
		"first apply multiple resources with status and prune": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
				testutil.Unstructured(t, resources["secret"]),
			},
			invInfo: inventoryInfo{
				name:      "inv-123",
				namespace: "default",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				ReconcileTimeout: time.Minute,
				InventoryPolicy:  inventory.PolicyMustMatch,
				EmitStatusEvents: true,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["secret"]),
					},
				},
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
						Status:     status.CurrentStatus,
					},
				},
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},

				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Started,
					},
				},
				// Secrets applied before Deployments (see pkg/ordering)
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created, // Create new
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created, // Create new
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				// Secrets before Deployments (see pkg/ordering)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				// Deployment before Secret (see statusEvents)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"apply multiple existing resources with status and prune": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
				testutil.Unstructured(t, resources["secret"]),
			},
			invInfo: inventoryInfo{
				name:      "inv-123",
				namespace: "default",
				id:        "test",
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(
						testutil.Unstructured(t, resources["deployment"]),
					),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
			},
			options: ApplierOptions{
				ReconcileTimeout: time.Minute,
				InventoryPolicy:  inventory.PolicyAdoptIfNoInventory,
				EmitStatusEvents: true,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["secret"]),
					},
				},
			},
			expectedStatusEvents: []testutil.ExpEvent{
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.CurrentStatus,
					},
				},
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},

				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Started,
					},
				},
				// Apply Secrets before Deployments (see ordering.SortableMetas)
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created, // Create new
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Configured, // Update existing
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				// Apply Secrets before Deployments (see ordering.SortableMetas)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				// Wait Deployments before Secrets (see testutil.GroupedEventsByID)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"apply no resources and prune all existing": {
			namespace: "default",
			resources: object.UnstructuredSet{},
			invInfo: inventoryInfo{
				name:      "inv-123",
				namespace: "default",
				id:        "test",
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(
						testutil.Unstructured(t, resources["deployment"]),
					),
					object.UnstructuredToObjMetadata(
						testutil.Unstructured(t, resources["secret"]),
					),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
				testutil.Unstructured(t, resources["secret"], testutil.AddOwningInv(t, "test")),
			},
			options: ApplierOptions{
				InventoryPolicy:  inventory.PolicyMustMatch,
				EmitStatusEvents: true,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.InProgressStatus,
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.NotFoundStatus,
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.NotFoundStatus,
					},
				},
			},
			expectedStatusEvents: []testutil.ExpEvent{
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
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
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
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
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.NotFoundStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Started,
					},
				},
				// Prune Deployments before Secrets (see ordering.SortableMetas)
				{
					EventType: event.PruneType,
					PruneEvent: &testutil.ExpPruneEvent{
						GroupName:  "prune-0",
						Operation:  event.Pruned,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.PruneType,
					PruneEvent: &testutil.ExpPruneEvent{
						GroupName:  "prune-0",
						Operation:  event.Pruned,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				// Prune Deployments before Secrets (see ordering.SortableMetas)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				// Wait Deployments before Secrets (see testutil.GroupedEventsByID)
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"apply resource with existing object belonging to different inventory": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
			},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "default",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "unmatched")),
			},
			options: ApplierOptions{
				ReconcileTimeout: time.Minute,
				InventoryPolicy:  inventory.PolicyMustMatch,
				EmitStatusEvents: true,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.CurrentStatus,
					},
				},
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
						Status:     status.CurrentStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Error:      testutil.EqualErrorType(inventory.NewInventoryOverlapError(fmt.Errorf(""))),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcileSkipped,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"resources belonging to a different inventory should not be pruned": {
			namespace: "default",
			resources: object.UnstructuredSet{},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "default",
				id:        "test",
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(
						testutil.Unstructured(t, resources["deployment"]),
					),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "unmatched")),
			},
			options: ApplierOptions{
				InventoryPolicy:  inventory.PolicyMustMatch,
				EmitStatusEvents: true,
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.PruneType,
					PruneEvent: &testutil.ExpPruneEvent{
						GroupName:  "prune-0",
						Operation:  event.PruneSkipped,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcileSkipped,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"prune with inventory object annotation matched": {
			namespace: "default",
			resources: object.UnstructuredSet{},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "default",
				id:        "test",
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(
						testutil.Unstructured(t, resources["deployment"]),
					),
				},
			},
			clusterObjs: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], testutil.AddOwningInv(t, "test")),
			},
			options: ApplierOptions{
				InventoryPolicy:  inventory.PolicyMustMatch,
				EmitStatusEvents: true,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.NotFoundStatus,
					},
				},
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
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.PruneType,
					PruneEvent: &testutil.ExpPruneEvent{
						GroupName:  "prune-0",
						Operation:  event.Pruned,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "prune-0",
						Action:    event.PruneAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"SkipInvalid - skip invalid objects and apply valid objects": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
					"$.metadata.name", "",
				}),
				testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
					"$.kind", "",
				}),
				testutil.Unstructured(t, resources["secret"]),
			},
			invInfo: inventoryInfo{
				name:      "inv-123",
				namespace: "default",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				ReconcileTimeout: time.Minute,
				InventoryPolicy:  inventory.PolicyAdoptIfNoInventory,
				EmitStatusEvents: true,
				ValidationPolicy: validation.SkipInvalid,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["secret"]),
					},
				},
			},
			expectedStatusEvents: []testutil.ExpEvent{
				{
					EventType: event.StatusType,
					StatusEvent: &testutil.ExpStatusEvent{
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
						Status:     status.CurrentStatus,
					},
				},
			},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.ValidationType,
					ValidationEvent: &testutil.ExpValidationEvent{
						Identifiers: object.ObjMetadataSet{
							object.UnstructuredToObjMetadata(
								testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
									"$.kind", "",
								}),
							),
						},
						Error: testutil.EqualErrorString(validation.NewError(
							field.Required(field.NewPath("kind"), "kind is required"),
							object.UnstructuredToObjMetadata(
								testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
									"$.kind", "",
								}),
							),
						).Error()),
					},
				},
				{
					EventType: event.ValidationType,
					ValidationEvent: &testutil.ExpValidationEvent{
						Identifiers: object.ObjMetadataSet{
							object.UnstructuredToObjMetadata(
								testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
									"$.metadata.name", "",
								}),
							),
						},
						Error: testutil.EqualErrorString(validation.NewError(
							field.Required(field.NewPath("metadata", "name"), "name is required"),
							object.UnstructuredToObjMetadata(
								testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
									"$.metadata.name", "",
								}),
							),
						).Error()),
					},
				},
				{
					EventType: event.InitType,
					InitEvent: &testutil.ExpInitEvent{},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-add-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},

				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Started,
					},
				},
				// Secret applied
				{
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created, // Create new
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "apply-0",
						Action:    event.ApplyAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Started,
					},
				},
				// Secret pending
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.ReconcilePending,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				// Secret reconciled
				{
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Operation:  event.Reconciled,
						Identifier: testutil.ToIdentifier(t, resources["secret"]),
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "wait-0",
						Action:    event.WaitAction,
						Type:      event.Finished,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Started,
					},
				},
				{
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						GroupName: "inventory-set-0",
						Action:    event.InventoryAction,
						Type:      event.Finished,
					},
				},
			},
		},
		"ExitEarly - exit early on invalid objects and skip valid objects": {
			namespace: "default",
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
					"$.metadata.name", "",
				}),
				testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
					"$.kind", "",
				}),
				testutil.Unstructured(t, resources["secret"]),
			},
			invInfo: inventoryInfo{
				name:      "inv-123",
				namespace: "default",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				ReconcileTimeout: time.Minute,
				InventoryPolicy:  inventory.PolicyAdoptIfNoInventory,
				EmitStatusEvents: true,
				ValidationPolicy: validation.ExitEarly,
			},
			statusEvents:         []pollevent.Event{},
			expectedStatusEvents: []testutil.ExpEvent{},
			expectedEvents: []testutil.ExpEvent{
				{
					EventType: event.ErrorType,
					ErrorEvent: &testutil.ExpErrorEvent{
						Err: testutil.EqualErrorString(multierror.New(
							validation.NewError(
								field.Required(field.NewPath("metadata", "name"), "name is required"),
								object.UnstructuredToObjMetadata(
									testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
										"$.metadata.name", "",
									}),
								),
							),
							validation.NewError(
								field.Required(field.NewPath("kind"), "kind is required"),
								object.UnstructuredToObjMetadata(
									testutil.Unstructured(t, resources["deployment"], JSONPathSetter{
										"$.kind", "",
									}),
								),
							),
						).Error()),
					},
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			poller := newFakePoller(tc.statusEvents)

			// Only feed valid objects into the TestApplier.
			// Invalid objects should not generate API requests.
			validObjs := object.UnstructuredSet{}
			for _, obj := range tc.resources {
				id := object.UnstructuredToObjMetadata(obj)
				if id.GroupKind.Kind == "" || id.Name == "" {
					continue
				}
				validObjs = append(validObjs, obj)
			}

			applier := newTestApplier(t,
				tc.invInfo,
				validObjs,
				tc.clusterObjs,
				poller,
			)

			// Context for Applier.Run
			runCtx, runCancel := context.WithCancel(context.Background())
			defer runCancel() // cleanup

			// Context for this test (in case Applier.Run never closes the event channel)
			testTimeout := 10 * time.Second
			testCtx, testCancel := context.WithTimeout(context.Background(), testTimeout)
			defer testCancel() // cleanup

			eventChannel := applier.Run(runCtx, tc.invInfo.toWrapped(), tc.resources, tc.options)

			// only start sending events once
			var once sync.Once

			var events []event.Event

		loop:
			for {
				select {
				case <-testCtx.Done():
					// Test timed out
					runCancel()
					if tc.expectTestTimeout {
						assert.Equal(t, context.DeadlineExceeded, testCtx.Err(), "Applier.Run failed to exit, but not because of expected timeout")
					} else {
						t.Errorf("Applier.Run failed to exit (timeout: %s)", testTimeout)
					}
					break loop

				case e, ok := <-eventChannel:
					if !ok {
						// Event channel closed
						testCancel()
						break loop
					}
					if e.Type == event.ActionGroupType &&
						e.ActionGroupEvent.Type == event.Finished {
						// Send events after the first apply/prune task ends
						if e.ActionGroupEvent.Action == event.ApplyAction ||
							e.ActionGroupEvent.Action == event.PruneAction {
							once.Do(func() {
								// start events
								poller.Start()
							})
						}
					}
					events = append(events, e)
				}
			}

			// Convert events to test events for comparison
			receivedEvents := testutil.EventsToExpEvents(events)

			// Validate & remove expected status events
			for _, e := range tc.expectedStatusEvents {
				var removed int
				receivedEvents, removed = testutil.RemoveEqualEvents(receivedEvents, e)
				if removed < 1 {
					t.Fatalf("Expected status event not received: %#v", e.StatusEvent)
				}
			}

			// sort to allow comparison of multiple apply/prune tasks in the same task group
			sort.Sort(testutil.GroupedEventsByID(receivedEvents))

			// Validate the rest of the events
			testutil.AssertEqual(t, tc.expectedEvents, receivedEvents,
				"Actual events (%d) do not match expected events (%d)",
				len(receivedEvents), len(tc.expectedEvents))

			// Validate that the expected timeout was the cause of the run completion.
			// just in case something else cancelled the run
			switch {
			case tc.expectRunTimeout:
				assert.Equal(t, context.DeadlineExceeded, runCtx.Err(), "Applier.Run exited, but not by expected context timeout")
			case tc.expectTestTimeout:
				assert.Equal(t, context.Canceled, runCtx.Err(), "Applier.Run exited, but not because of expected context cancellation")
			default:
				assert.Nil(t, runCtx.Err(), "Applier.Run exited, but context error is not nil")
			}
		})
	}
}

func TestApplierCancel(t *testing.T) {
	testCases := map[string]struct {
		// resources input to applier
		resources object.UnstructuredSet
		// inventory input to applier
		invInfo inventoryInfo
		// objects in the cluster
		clusterObjs object.UnstructuredSet
		// options input to applier.Run
		options ApplierOptions
		// timeout for applier.Run
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
		"cancelled by caller while waiting for reconcile": {
			expectRunTimeout: true,
			runTimeout:       2 * time.Second,
			testTimeout:      30 * time.Second,
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
			},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "test",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				// EmitStatusEvents required to test event output
				EmitStatusEvents: true,
				NoPrune:          true,
				InventoryPolicy:  inventory.PolicyMustMatch,
				// ReconcileTimeout required to enable WaitTasks
				ReconcileTimeout: 1 * time.Minute,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				// Resource never becomes Current, blocking applier.Run from exiting
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
					// InvAddTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-add-0",
						Type:      event.Started,
					},
				},
				{
					// InvAddTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-add-0",
						Type:      event.Finished,
					},
				},
				{
					// ApplyTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.ApplyAction,
						GroupName: "apply-0",
						Type:      event.Started,
					},
				},
				{
					// Apply Deployment
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					// ApplyTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.ApplyAction,
						GroupName: "apply-0",
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
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Operation:  event.ReconcilePending,
					},
				},
				// Deployment never becomes Current.
				// WaitTask is expected to be cancelled before ReconcileTimeout.
				// Cancelled WaitTask do not sent individual timeout WaitEvents
				{
					// WaitTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.WaitAction,
						GroupName: "wait-0",
						Type:      event.Finished, // TODO: add Cancelled event type
					},
				},
				// TODO: Update the inventory after cancellation
				// {
				// 	// InvSetTask start
				// 	EventType: event.ActionGroupType,
				// 	ActionGroupEvent: &testutil.ExpActionGroupEvent{
				// 		Action:    event.InventoryAction,
				// 		GroupName: "inventory-set-0",
				// 		Type:      event.Started,
				// 	},
				// },
				// {
				// 	// InvSetTask finished
				// 	EventType: event.ActionGroupType,
				// 	ActionGroupEvent: &testutil.ExpActionGroupEvent{
				// 		Action:    event.InventoryAction,
				// 		GroupName: "inventory-set-0",
				// 		Type:      event.Finished,
				// 	},
				// },
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
			resources: object.UnstructuredSet{
				testutil.Unstructured(t, resources["deployment"]),
			},
			invInfo: inventoryInfo{
				name:      "abc-123",
				namespace: "test",
				id:        "test",
			},
			clusterObjs: object.UnstructuredSet{},
			options: ApplierOptions{
				// EmitStatusEvents required to test event output
				EmitStatusEvents: true,
				NoPrune:          true,
				InventoryPolicy:  inventory.PolicyMustMatch,
				// ReconcileTimeout required to enable WaitTasks
				ReconcileTimeout: 1 * time.Minute,
			},
			statusEvents: []pollevent.Event{
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.InProgressStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Status:     status.CurrentStatus,
						Resource:   testutil.Unstructured(t, resources["deployment"]),
					},
				},
				// Resource becoming Current should unblock applier.Run WaitTask
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
						Status:     status.CurrentStatus,
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
					// InvAddTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-add-0",
						Type:      event.Started,
					},
				},
				{
					// InvAddTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-add-0",
						Type:      event.Finished,
					},
				},
				{
					// ApplyTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.ApplyAction,
						GroupName: "apply-0",
						Type:      event.Started,
					},
				},
				{
					// Apply Deployment
					EventType: event.ApplyType,
					ApplyEvent: &testutil.ExpApplyEvent{
						GroupName:  "apply-0",
						Operation:  event.Created,
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
					},
				},
				{
					// ApplyTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.ApplyAction,
						GroupName: "apply-0",
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
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Operation:  event.ReconcilePending,
					},
				},
				{
					// Deployment becomes Current.
					EventType: event.WaitType,
					WaitEvent: &testutil.ExpWaitEvent{
						GroupName:  "wait-0",
						Identifier: testutil.ToIdentifier(t, resources["deployment"]),
						Operation:  event.Reconciled,
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
					// InvSetTask start
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-set-0",
						Type:      event.Started,
					},
				},
				{
					// InvSetTask finished
					EventType: event.ActionGroupType,
					ActionGroupEvent: &testutil.ExpActionGroupEvent{
						Action:    event.InventoryAction,
						GroupName: "inventory-set-0",
						Type:      event.Finished,
					},
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			poller := newFakePoller(tc.statusEvents)

			applier := newTestApplier(t,
				tc.invInfo,
				tc.resources,
				tc.clusterObjs,
				poller,
			)

			// Context for Applier.Run
			runCtx, runCancel := context.WithTimeout(context.Background(), tc.runTimeout)
			defer runCancel() // cleanup

			// Context for this test (in case Applier.Run never closes the event channel)
			testCtx, testCancel := context.WithTimeout(context.Background(), tc.testTimeout)
			defer testCancel() // cleanup

			eventChannel := applier.Run(runCtx, tc.invInfo.toWrapped(), tc.resources, tc.options)

			// only start sending events once
			var once sync.Once

			var events []event.Event

		loop:
			for {
				select {
				case <-testCtx.Done():
					// Test timed out
					runCancel()
					t.Errorf("Applier.Run failed to respond to cancellation (expected: %s, timeout: %s)", tc.runTimeout, tc.testTimeout)
					break loop

				case e, ok := <-eventChannel:
					if !ok {
						// Event channel closed
						testCancel()
						break loop
					}
					events = append(events, e)

					if e.Type == event.ActionGroupType &&
						e.ActionGroupEvent.Type == event.Finished {
						// Send events after the first apply/prune task ends
						if e.ActionGroupEvent.Action == event.ApplyAction ||
							e.ActionGroupEvent.Action == event.PruneAction {
							once.Do(func() {
								// start events
								poller.Start()
							})
						}
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
					t.Fatalf("Expected status event not received: %#v", e.StatusEvent)
				}
			}

			// Validate the rest of the events
			testutil.AssertEqual(t, tc.expectedEvents, receivedEvents,
				"Actual events (%d) do not match expected events (%d)",
				len(receivedEvents), len(tc.expectedEvents))

			// Validate that the expected timeout was the cause of the run completion.
			// just in case something else cancelled the run
			if tc.expectRunTimeout {
				assert.Equal(t, context.DeadlineExceeded, runCtx.Err(), "Applier.Run exited, but not by expected timeout")
			} else {
				assert.NoError(t, runCtx.Err(), "Applier.Run exited, but not by expected timeout")
			}
		})
	}
}

func TestReadAndPrepareObjectsNilInv(t *testing.T) {
	applier := Applier{}
	_, _, err := applier.prepareObjects(nil, object.UnstructuredSet{}, ApplierOptions{})
	assert.Error(t, err)
}

func TestReadAndPrepareObjects(t *testing.T) {
	inventoryObj := testutil.Unstructured(t, resources["inventory"])
	inventory := inventory.WrapInventoryInfoObj(inventoryObj)

	obj1 := testutil.Unstructured(t, resources["obj1"])
	obj2 := testutil.Unstructured(t, resources["obj2"])
	clusterScopedObj := testutil.Unstructured(t, resources["clusterScopedObj"])

	testCases := map[string]struct {
		// objects in the cluster
		clusterObjs object.UnstructuredSet
		// inventory input to applier
		invInfo inventoryInfo
		// resources input to applier
		resources object.UnstructuredSet
		// expected objects to apply
		applyObjs object.UnstructuredSet
		// expected objects to prune
		pruneObjs object.UnstructuredSet
		// expected error
		isError bool
	}{
		"objects include inventory": {
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
			},
			resources: object.UnstructuredSet{inventoryObj},
			isError:   true,
		},
		"empty inventory, empty objects, apply none, prune none": {
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
			},
		},
		"one in inventory, empty objects, prune one": {
			clusterObjs: object.UnstructuredSet{obj1},
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(obj1),
				},
			},
			pruneObjs: object.UnstructuredSet{obj1},
		},
		"all in inventory, apply all": {
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(obj1),
					object.UnstructuredToObjMetadata(clusterScopedObj),
				},
			},
			resources: object.UnstructuredSet{obj1, clusterScopedObj},
			applyObjs: object.UnstructuredSet{obj1, clusterScopedObj},
		},
		"disjoint set, apply new, prune old": {
			clusterObjs: object.UnstructuredSet{obj2},
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(obj2),
				},
			},
			resources: object.UnstructuredSet{obj1, clusterScopedObj},
			applyObjs: object.UnstructuredSet{obj1, clusterScopedObj},
			pruneObjs: object.UnstructuredSet{obj2},
		},
		"most in inventory, apply all": {
			clusterObjs: object.UnstructuredSet{obj2},
			invInfo: inventoryInfo{
				name:      inventory.Name(),
				namespace: inventory.Namespace(),
				id:        inventory.ID(),
				set: object.ObjMetadataSet{
					object.UnstructuredToObjMetadata(obj2),
				},
			},
			resources: object.UnstructuredSet{obj1, obj2, clusterScopedObj},
			applyObjs: object.UnstructuredSet{obj1, obj2, clusterScopedObj},
			pruneObjs: object.UnstructuredSet{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			applier := newTestApplier(t,
				tc.invInfo,
				tc.resources,
				tc.clusterObjs,
				// no events needed for prepareObjects
				newFakePoller([]pollevent.Event{}),
			)

			applyObjs, pruneObjs, err := applier.prepareObjects(tc.invInfo.toWrapped(), tc.resources, ApplierOptions{})
			if tc.isError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			testutil.AssertEqual(t, applyObjs, tc.applyObjs,
				"Actual applied objects (%d) do not match expected applied objects (%d)",
				len(applyObjs), len(tc.applyObjs))

			testutil.AssertEqual(t, pruneObjs, tc.pruneObjs,
				"Actual pruned objects (%d) do not match expected pruned objects (%d)",
				len(pruneObjs), len(tc.pruneObjs))
		})
	}
}
