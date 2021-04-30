// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

var (
	inventoryTemplate = `
kind: ConfigMap
apiVersion: v1
metadata:
  labels:
    cli-utils.sigs.k8s.io/inventory-id: test
  name: foo
  namespace: default
`
	depObject = object.ObjMetadata{
		Name:      "foo",
		Namespace: "default",
		GroupKind: schema.GroupKind{
			Group: "apps",
			Kind:  "Deployment",
		},
	}

	stsObject = object.ObjMetadata{
		Name:      "bar",
		Namespace: "default",
		GroupKind: schema.GroupKind{
			Group: "apps",
			Kind:  "StatefulSet",
		},
	}
)

func TestStatusCommand(t *testing.T) {
	testCases := map[string]struct {
		pollUntil      string
		printer        string
		timeout        time.Duration
		input          string
		inventory      []object.ObjMetadata
		events         []pollevent.Event
		expectedErrMsg string
		expectedOutput string
	}{
		"no inventory template": {
			input:          "",
			expectedErrMsg: "Package uninitialized. Please run \"init\" command.",
		},
		"no inventory in live state": {
			input:          inventoryTemplate,
			expectedOutput: "no resources found in the inventory\n",
		},
		"wait for all known": {
			pollUntil: "known",
			printer:   "events",
			input:     inventoryTemplate,
			inventory: []object.ObjMetadata{
				depObject,
				stsObject,
			},
			events: []pollevent.Event{
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
			},
			expectedOutput: `
deployment.apps/foo is InProgress: inProgress
statefulset.apps/bar is Current: current
`,
		},
		"wait for all current": {
			pollUntil: "current",
			printer:   "events",
			input:     inventoryTemplate,
			inventory: []object.ObjMetadata{
				depObject,
				stsObject,
			},
			events: []pollevent.Event{
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
			},
			expectedOutput: `
deployment.apps/foo is InProgress: inProgress
statefulset.apps/bar is InProgress: inProgress
statefulset.apps/bar is Current: current
deployment.apps/foo is Current: current
`,
		},
		"wait for all deleted": {
			pollUntil: "deleted",
			printer:   "events",
			input:     inventoryTemplate,
			inventory: []object.ObjMetadata{
				depObject,
				stsObject,
			},
			events: []pollevent.Event{
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.NotFoundStatus,
						Message:    "notFound",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.NotFoundStatus,
						Message:    "notFound",
					},
				},
			},
			expectedOutput: `
statefulset.apps/bar is NotFound: notFound
deployment.apps/foo is NotFound: notFound
`,
		},
		"forever with timeout": {
			pollUntil: "forever",
			printer:   "events",
			timeout:   2 * time.Second,
			input:     inventoryTemplate,
			inventory: []object.ObjMetadata{
				depObject,
				stsObject,
			},
			events: []pollevent.Event{
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					EventType: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
			},
			expectedOutput: `
statefulset.apps/bar is InProgress: inProgress
deployment.apps/foo is InProgress: inProgress
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			tf := cmdtesting.NewTestFactory().WithNamespace("namespace")
			defer tf.Cleanup()

			provider := provider.NewFakeProvider(tf, tc.inventory)
			loader := manifestreader.NewFakeLoader(tf, tc.inventory)
			runner := &StatusRunner{
				provider: provider,
				loader:   loader,
				pollerFactoryFunc: func(c cmdutil.Factory) (poller.Poller, error) {
					return &fakePoller{tc.events}, nil
				},

				pollUntil: tc.pollUntil,
				output:    tc.printer,
				timeout:   tc.timeout,
			}

			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(tc.input))
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			err := runner.runE(cmd, []string{})

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(buf.String()), strings.TrimSpace(tc.expectedOutput))
		})
	}
}

type fakePoller struct {
	events []pollevent.Event
}

func (f *fakePoller) Poll(ctx context.Context, _ []object.ObjMetadata,
	_ polling.Options) <-chan pollevent.Event {
	eventChannel := make(chan pollevent.Event)
	go func() {
		defer close(eventChannel)
		for _, e := range f.events {
			eventChannel <- e
		}
		<-ctx.Done()
	}()
	return eventChannel
}
