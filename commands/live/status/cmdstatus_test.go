// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package status

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
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
			expectedErrMsg: "no ResourceGroup object was provided within the stream or package",
		},
		"invalid value for pollUntil": {
			pollUntil:      "doesNotExist",
			expectedErrMsg: "pollUntil must be one of known,current,deleted,forever",
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
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
			},
			expectedOutput: `
foo/deployment.apps/default/foo is InProgress: inProgress
foo/statefulset.apps/default/bar is Current: current
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
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.CurrentStatus,
						Message:    "current",
					},
				},
			},
			expectedOutput: `
foo/deployment.apps/default/foo is InProgress: inProgress
foo/statefulset.apps/default/bar is InProgress: inProgress
foo/statefulset.apps/default/bar is Current: current
foo/deployment.apps/default/foo is Current: current
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
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.NotFoundStatus,
						Message:    "notFound",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.NotFoundStatus,
						Message:    "notFound",
					},
				},
			},
			expectedOutput: `
foo/statefulset.apps/default/bar is NotFound: notFound
foo/deployment.apps/default/foo is NotFound: notFound
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
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: stsObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
				{
					Type: pollevent.ResourceUpdateEvent,
					Resource: &pollevent.ResourceStatus{
						Identifier: depObject,
						Status:     status.InProgressStatus,
						Message:    "inProgress",
					},
				},
			},
			expectedOutput: `
foo/statefulset.apps/default/bar is InProgress: inProgress
foo/deployment.apps/default/foo is InProgress: inProgress
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
			ctx := fake.CtxWithPrinter(&outBuf, &outBuf)
			invFactory := inventory.FakeClientFactory(tc.inventory)
			loader := NewFakeLoader(ctx, tf, tc.inventory)
			runner := NewRunner(ctx, tf, invFactory, loader)
			runner.PollerFactoryFunc = func(c cmdutil.Factory) (poller.Poller, error) {
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
			runner.Command.SetOut(&outBuf)
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

func (f *fakePoller) Poll(ctx context.Context, _ object.ObjMetadataSet,
	_ polling.PollOptions) <-chan pollevent.Event {
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
