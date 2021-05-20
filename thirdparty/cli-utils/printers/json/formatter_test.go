// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package json

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/print/list"
)

func TestFormatter_FormatApplyEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.ApplyEvent
		applyStats      *list.ApplyStats
		statusCollector list.Collector
		expected        []map[string]interface{}
	}{
		"resource created without dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.ApplyEvent{
				Operation:  event.Created,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: []map[string]interface{}{
				{
					"eventType": "resourceApplied",
					"group":     "apps",
					"kind":      "Deployment",
					"name":      "my-dep",
					"namespace": "default",
					"operation": "Created",
					"timestamp": "",
					"type":      "apply",
				},
			},
		},
		"resource updated with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: []map[string]interface{}{
				{
					"eventType": "resourceApplied",
					"group":     "apps",
					"kind":      "Deployment",
					"name":      "my-dep",
					"namespace": "",
					"operation": "Configured",
					"timestamp": "",
					"type":      "apply",
				},
			},
		},
		"resource updated with server dryrun": {
			dryRunStrategy: common.DryRunServer,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("batch", "CronJob", "foo", "my-cron"),
			},
			expected: []map[string]interface{}{
				{
					"eventType": "resourceApplied",
					"group":     "batch",
					"kind":      "CronJob",
					"name":      "my-cron",
					"namespace": "foo",
					"operation": "Configured",
					"timestamp": "",
					"type":      "apply",
				},
			},
		},
		"completed event": {
			dryRunStrategy: common.DryRunNone,
			event: event.ApplyEvent{
				Type: event.ApplyEventCompleted,
			},
			applyStats: &list.ApplyStats{
				ServersideApplied: 1,
			},
			statusCollector: &fakeCollector{
				m: map[object.ObjMetadata]event.StatusEvent{
					{ //nolint:gofmt
						GroupKind: schema.GroupKind{
							Group: "apps",
							Kind:  "Deployment",
						},
						Namespace: "foo",
						Name:      "my-dep",
					}: {
						Resource: &pollevent.ResourceStatus{
							Status:  status.CurrentStatus,
							Message: "Resource is Current",
						},
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"configuredCount": 0,
					"count":           1,
					"createdCount":    0,
					"eventType":       "completed",
					"failedCount":     0,
					"serverSideCount": 1,
					"type":            "apply",
					"unchangedCount":  0,
					"timestamp":       "",
				},
				{
					"eventType": "resourceStatus",
					"group":     "apps",
					"kind":      "Deployment",
					"message":   "Resource is Current",
					"name":      "my-dep",
					"namespace": "foo",
					"status":    "Current",
					"timestamp": "",
					"type":      "status",
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatApplyEvent(tc.event, tc.applyStats, tc.statusCollector)
			assert.NoError(t, err)

			objects := strings.Split(strings.TrimSpace(out.String()), "\n")

			if !assert.Equal(t, len(tc.expected), len(objects)) {
				t.FailNow()
			}
			for i := range tc.expected {
				assertOutput(t, tc.expected[i], objects[i])
			}
		})
	}
}

func TestFormatter_FormatStatusEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.StatusEvent
		statusCollector list.Collector
		expected        map[string]interface{}
	}{
		"resource update with Current status": {
			dryRunStrategy: common.DryRunNone,
			event: event.StatusEvent{
				Type: event.StatusEventResourceUpdate,
				Resource: &pollevent.ResourceStatus{
					Identifier: object.ObjMetadata{
						GroupKind: schema.GroupKind{
							Group: "apps",
							Kind:  "Deployment",
						},
						Namespace: "foo",
						Name:      "bar",
					},
					Status:  status.CurrentStatus,
					Message: "Resource is Current",
				},
			},
			expected: map[string]interface{}{
				"eventType": "resourceStatus",
				"group":     "apps",
				"kind":      "Deployment",
				"message":   "Resource is Current",
				"name":      "bar",
				"namespace": "foo",
				"status":    "Current",
				"timestamp": "",
				"type":      "status",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatStatusEvent(tc.event, tc.statusCollector)
			assert.NoError(t, err)

			assertOutput(t, tc.expected, out.String())
		})
	}
}

func TestFormatter_FormatPruneEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy common.DryRunStrategy
		event          event.PruneEvent
		pruneStats     *list.PruneStats
		expected       map[string]interface{}
	}{
		"resource pruned without dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Operation:  event.Pruned,
				Type:       event.PruneEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: map[string]interface{}{
				"eventType": "resourcePruned",
				"group":     "apps",
				"kind":      "Deployment",
				"name":      "my-dep",
				"namespace": "default",
				"operation": "Pruned",
				"timestamp": "",
				"type":      "prune",
			},
		},
		"resource skipped with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.PruneEvent{
				Operation:  event.PruneSkipped,
				Type:       event.PruneEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: map[string]interface{}{
				"eventType": "resourcePruned",
				"group":     "apps",
				"kind":      "Deployment",
				"name":      "my-dep",
				"namespace": "",
				"operation": "PruneSkipped",
				"timestamp": "",
				"type":      "prune",
			},
		},
		"prune event with completed status": {
			dryRunStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Type: event.PruneEventCompleted,
			},
			pruneStats: &list.PruneStats{
				Pruned:  1,
				Skipped: 2,
			},
			expected: map[string]interface{}{
				"eventType": "completed",
				"skipped":   2,
				"pruned":    1,
				"timestamp": "",
				"type":      "prune",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatPruneEvent(tc.event, tc.pruneStats)
			assert.NoError(t, err)

			assertOutput(t, tc.expected, out.String())
		})
	}
}

func TestFormatter_FormatDeleteEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.DeleteEvent
		deleteStats     *list.DeleteStats
		statusCollector list.Collector
		expected        map[string]interface{}
	}{
		"resource deleted without no dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.DeleteEvent{
				Operation:  event.Deleted,
				Type:       event.DeleteEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: map[string]interface{}{
				"eventType": "resourceDeleted",
				"group":     "apps",
				"kind":      "Deployment",
				"name":      "my-dep",
				"namespace": "default",
				"operation": "Deleted",
				"timestamp": "",
				"type":      "delete",
			},
		},
		"resource skipped with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.DeleteEvent{
				Operation:  event.DeleteSkipped,
				Type:       event.DeleteEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: map[string]interface{}{
				"eventType": "resourceDeleted",
				"group":     "apps",
				"kind":      "Deployment",
				"name":      "my-dep",
				"namespace": "",
				"operation": "DeleteSkipped",
				"timestamp": "",
				"type":      "delete",
			},
		},
		"delete event with completed status": {
			dryRunStrategy: common.DryRunNone,
			event: event.DeleteEvent{
				Type: event.DeleteEventCompleted,
			},
			deleteStats: &list.DeleteStats{
				Deleted: 1,
				Skipped: 2,
			},
			expected: map[string]interface{}{
				"deleted":   1,
				"eventType": "completed",
				"skipped":   2,
				"timestamp": "",
				"type":      "delete",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatDeleteEvent(tc.event, tc.deleteStats)
			assert.NoError(t, err)

			assertOutput(t, tc.expected, out.String())
		})
	}
}

// nolint:unparam
func assertOutput(t *testing.T, expectedMap map[string]interface{}, actual string) bool {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(actual), &m)
	if !assert.NoError(t, err) {
		return false
	}

	if _, found := expectedMap["timestamp"]; found {
		if _, ok := m["timestamp"]; ok {
			delete(expectedMap, "timestamp")
			delete(m, "timestamp")
		} else {
			t.Error("expected to find key 'timestamp', but didn't")
			return false
		}
	}

	for key, val := range m {
		if floatVal, ok := val.(float64); ok {
			m[key] = int(floatVal)
		}
	}

	return assert.Equal(t, expectedMap, m)
}

func createIdentifier(group, kind, namespace, name string) object.ObjMetadata {
	return object.ObjMetadata{
		Namespace: namespace,
		Name:      name,
		GroupKind: schema.GroupKind{
			Group: group,
			Kind:  kind,
		},
	}
}

type fakeCollector struct {
	m map[object.ObjMetadata]event.StatusEvent
}

func (f *fakeCollector) LatestStatus() map[object.ObjMetadata]event.StatusEvent {
	return f.m
}
