// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/printer/fake"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdtesting "k8s.io/kubectl/pkg/cmd/testing"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply/poller"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var (
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
		kptfileInv     *kptfilev1.Inventory
		inventory      []object.ObjMetadata
		events         []pollevent.Event
		expectedErrMsg string
		expectedOutput string
	}{
		"no inventory template": {
			kptfileInv:     nil,
			expectedErrMsg: "inventory failed validation",
		},
		"invalid value for pollUntil": {
			pollUntil:      "doesNotExist",
			expectedErrMsg: "pollUntil must be one of \"known\", \"current\", \"deleted\", \"forever\"",
		},
		"no inventory in live state": {
			kptfileInv: &kptfilev1.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "test",
			},
			expectedOutput: "no resources found in the inventory\n",
		},
		"wait for all known": {
			pollUntil: "known",
			printer:   "events",
			kptfileInv: &kptfilev1.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "test",
			},
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
			kptfileInv: &kptfilev1.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "test",
			},
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
			kptfileInv: &kptfilev1.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "test",
			},
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
			kptfileInv: &kptfilev1.Inventory{
				Name:        "foo",
				Namespace:   "default",
				InventoryID: "test",
			},
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

			w, clean := testutil.SetupWorkspace(t)
			defer clean()
			kf := kptfileutil.DefaultKptfile(filepath.Base(w.WorkspaceDirectory))
			kf.Inventory = tc.kptfileInv
			testutil.AddKptfileToWorkspace(t, w, kf)

			revert := testutil.Chdir(t, w.WorkspaceDirectory)
			defer revert()

			var outBuf bytes.Buffer
			runner := NewRunner(fake.CtxWithPrinter(&outBuf, &outBuf), tf)
			runner.invClientFunc = func(f cmdutil.Factory) (inventory.InventoryClient, error) {
				return inventory.NewFakeInventoryClient(tc.inventory), nil
			}
			runner.pollerFactoryFunc = func(c cmdutil.Factory) (poller.Poller, error) {
				return &fakePoller{tc.events}, nil
			}

			args := []string{}
			if tc.pollUntil != "" {
				args = append(args, []string{
					"--poll-until", tc.pollUntil,
				}...)
			}
			if tc.timeout != time.Duration(0) {
				args = append(args, []string{
					"--timeout", tc.timeout.String(),
				}...)
			}
			runner.Command.SetArgs(args)
			err := runner.Command.Execute()

			if tc.expectedErrMsg != "" {
				if !assert.Error(t, err) {
					t.FailNow()
				}
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, strings.TrimSpace(tc.expectedOutput), strings.TrimSpace(outBuf.String()))
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
