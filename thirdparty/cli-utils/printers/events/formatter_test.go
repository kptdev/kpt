// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/thirdparty/cli-utils/print/list"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestFormatter_FormatApplyEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.ApplyEvent
		applyStats      *list.ApplyStats
		statusCollector list.Collector
		expected        string
	}{
		"resource created without no dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.ApplyEvent{
				Operation:  event.Created,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep created",
		},
		"resource updated with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep configured (dry-run)",
		},
		"resource updated with server dryrun": {
			dryRunStrategy: common.DryRunServer,
			event: event.ApplyEvent{
				Operation:  event.Configured,
				Identifier: createIdentifier("batch", "CronJob", "foo", "my-cron"),
			},
			expected: "cronjob.batch/my-cron configured (dry-run-server)",
		},
		"apply event with error should display the error": {
			dryRunStrategy: common.DryRunServer,
			event: event.ApplyEvent{
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test error"),
			},
			expected: "deployment.apps/my-dep apply failed: this is a test error (dry-run-server)",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatApplyEvent(tc.event)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatStatusEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.StatusEvent
		statusCollector list.Collector
		expected        string
	}{
		"resource update with Current status": {
			dryRunStrategy: common.DryRunNone,
			event: event.StatusEvent{
				Identifier: object.ObjMetadata{
					GroupKind: schema.GroupKind{
						Group: "apps",
						Kind:  "Deployment",
					},
					Namespace: "foo",
					Name:      "bar",
				},
				PollResourceInfo: &pollevent.ResourceStatus{
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
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatStatusEvent(tc.event)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatPruneEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy common.DryRunStrategy
		event          event.PruneEvent
		pruneStats     *list.PruneStats
		expected       string
	}{
		"resource pruned without no dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Operation:  event.Pruned,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep pruned",
		},
		"resource skipped with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.PruneEvent{
				Operation:  event.PruneSkipped,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep prune skipped (dry-run)",
		},
		"resource with prune error": {
			dryRunStrategy: common.DryRunNone,
			event: event.PruneEvent{
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test"),
			},
			expected: "deployment.apps/my-dep prune failed: this is a test",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatPruneEvent(tc.event)
			assert.NoError(t, err)

			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(out.String()))
		})
	}
}

func TestFormatter_FormatDeleteEvent(t *testing.T) {
	testCases := map[string]struct {
		dryRunStrategy  common.DryRunStrategy
		event           event.DeleteEvent
		deleteStats     *list.DeleteStats
		statusCollector list.Collector
		expected        string
	}{
		"resource deleted without no dryrun": {
			dryRunStrategy: common.DryRunNone,
			event: event.DeleteEvent{
				Operation:  event.Deleted,
				Identifier: createIdentifier("apps", "Deployment", "default", "my-dep"),
				Object:     createObject("apps", "Deployment", "default", "my-dep"),
			},
			expected: "deployment.apps/my-dep deleted",
		},
		"resource skipped with client dryrun": {
			dryRunStrategy: common.DryRunClient,
			event: event.DeleteEvent{
				Operation:  event.DeleteSkipped,
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Object:     createObject("apps", "Deployment", "", "my-dep"),
			},
			expected: "deployment.apps/my-dep delete skipped (dry-run)",
		},
		"resource with delete error": {
			dryRunStrategy: common.DryRunServer,
			event: event.DeleteEvent{
				Object:     createObject("apps", "Deployment", "", "my-dep"),
				Identifier: createIdentifier("apps", "Deployment", "", "my-dep"),
				Error:      fmt.Errorf("this is a test"),
			},
			expected: "deployment.apps/my-dep deletion failed: this is a test (dry-run-server)",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			ioStreams, _, out, _ := genericclioptions.NewTestIOStreams() //nolint:dogsled
			formatter := NewFormatter(ioStreams, tc.dryRunStrategy)
			err := formatter.FormatDeleteEvent(tc.event)
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
