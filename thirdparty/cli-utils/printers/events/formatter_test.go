// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		previewStrategy common.DryRunStrategy
		event           event.ApplyEvent
		applyStats      *list.ApplyStats
		statusCollector list.Collector
		expected        string
	}{
		"resource created without no dryrun": {
			previewStrategy: common.DryRunNone,
			event: event.ApplyEvent{
				Operation:  event.Created,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep created",
		},
		"resource updated with client dryrun": {
			previewStrategy: common.DryRunClient,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep configured (preview)",
		},
		"resource updated with server dryrun": {
			previewStrategy: common.DryRunServer,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("batch", "CronJob", "foo", "my-cron"),
			},
			expected: "cronjob.batch/my-cron configured (preview-server)",
		},
		"apply event with error should display the error": {
			previewStrategy: common.DryRunServer,
			event: event.ApplyEvent{
				Operation:  event.Failed,
				Type:       event.ApplyEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test error"),
			},
			expected: "deployment.apps/my-dep failed: this is a test error (preview-server)",
		},
		"completed event": {
			previewStrategy: common.DryRunNone,
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
			expected: `
1 resource(s) applied. 0 created, 0 unchanged, 0 configured, 0 failed, 1 serverside applied
deployment.apps/my-dep is Current: Resource is Current
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.previewStrategy)
			err := formatter.FormatApplyEvent(tc.event, tc.applyStats, tc.statusCollector)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatStatusEvent(t *testing.T) {
	testCases := map[string]struct {
		previewStrategy common.DryRunStrategy
		event           event.StatusEvent
		statusCollector list.Collector
		expected        string
	}{
		"resource update with Current status": {
			previewStrategy: common.DryRunNone,
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
			expected: "deployment.apps/bar is Current: Resource is Current",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.previewStrategy)
			err := formatter.FormatStatusEvent(tc.event, tc.statusCollector)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatPruneEvent(t *testing.T) {
	testCases := map[string]struct {
		previewStrategy common.DryRunStrategy
		event           event.PruneEvent
		pruneStats      *list.PruneStats
		expected        string
	}{
		"resource pruned without no dryrun": {
			previewStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Operation:  event.Pruned,
				Type:       event.PruneEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep pruned",
		},
		"resource skipped with client dryrun": {
			previewStrategy: common.DryRunClient,
			event: event.PruneEvent{
				Operation:  event.PruneSkipped,
				Type:       event.PruneEventResourceUpdate,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep prune skipped (preview)",
		},
		"resource with prune error": {
			previewStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Type:       event.PruneEventFailed,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test"),
			},
			expected: "deployment.apps/my-dep prune failed: this is a test",
		},
		"prune event with completed status": {
			previewStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Type: event.PruneEventCompleted,
			},
			pruneStats: &list.PruneStats{
				Pruned:  1,
				Skipped: 2,
			},
			expected: "1 resource(s) pruned, 2 skipped, 0 failed",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.previewStrategy)
			err := formatter.FormatPruneEvent(tc.event, tc.pruneStats)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatDeleteEvent(t *testing.T) {
	testCases := map[string]struct {
		previewStrategy common.DryRunStrategy
		event           event.DeleteEvent
		deleteStats     *list.DeleteStats
		statusCollector list.Collector
		expected        string
	}{
		"resource deleted without no dryrun": {
			previewStrategy: common.DryRunNone,
			event: event.DeleteEvent{
				Operation: event.Deleted,
				Type:      event.DeleteEventResourceUpdate,
				Object:    createObject("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep deleted",
		},
		"resource skipped with client dryrun": {
			previewStrategy: common.DryRunClient,
			event: event.DeleteEvent{
				Operation: event.DeleteSkipped,
				Type:      event.DeleteEventResourceUpdate,
				Object:    createObject("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep delete skipped (preview)",
		},
		"resource with delete error": {
			previewStrategy: common.DryRunServer,
			event: event.DeleteEvent{
				Type:       event.DeleteEventFailed,
				Object:     createObject("apps", "Deployment", "", "my-dep"),
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test"),
			},
			expected: "deployment.apps/my-dep deletion failed: this is a test (preview-server)",
		},
		"delete event with completed status": {
			previewStrategy: common.DryRunNone,
			event: event.DeleteEvent{
				Type: event.DeleteEventCompleted,
			},
			deleteStats: &list.DeleteStats{
				Deleted: 1,
				Skipped: 2,
			},
			expected: "1 resource(s) deleted, 2 skipped",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.previewStrategy)
			err := formatter.FormatDeleteEvent(tc.event, tc.deleteStats)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func createObject(group, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/v1", group),
			"kind":       kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
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
