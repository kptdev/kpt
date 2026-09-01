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

package planner

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/testutil"
)

var (
	deploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: foo
  namespace: default
spec:
  replicas: 1
`
)

func TestClusterPlanner(t *testing.T) {
	testCases := map[string]struct {
		resources        []*unstructured.Unstructured
		clusterResources []*unstructured.Unstructured
		events           []event.Event

		expectedPlan *Plan
	}{
		"single new resource": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.ApplyAction,
								Name:   "apply-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.ApplyType,
					ApplyEvent: event.ApplyEvent{
						GroupName:  "apply-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.ApplySuccessful,
						Resource:   testutil.Unstructured(t, deploymentYAML),
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:      Create,
						Name:      "foo",
						Namespace: "default",
						Group:     "apps",
						Kind:      "Deployment",
						Updated:   testutil.Unstructured(t, deploymentYAML),
					},
				},
			},
		},
		"skipped resource with reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.ApplyAction,
								Name:   "apply-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.ApplyType,
					ApplyEvent: event.ApplyEvent{
						GroupName:  "apply-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.ApplySkipped,
						Error:      fmt.Errorf("some skip reason"),
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:       Skip,
						Name:       "foo",
						Namespace:  "default",
						Group:      "apps",
						Kind:       "Deployment",
						SkipReason: "some skip reason",
					},
				},
			},
		},
		"skipped apply resource without reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.ApplyAction,
								Name:   "apply-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.ApplyType,
					ApplyEvent: event.ApplyEvent{
						GroupName:  "apply-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.ApplySkipped,
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:      Skip,
						Name:      "foo",
						Namespace: "default",
						Group:     "apps",
						Kind:      "Deployment",
					},
				},
			},
		},
		"skipped prune resource with reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.PruneAction,
								Name:   "prune-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.PruneType,
					PruneEvent: event.PruneEvent{
						GroupName:  "prune-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.PruneSkipped,
						Error:      fmt.Errorf("some skip reason"),
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:       Skip,
						Name:       "foo",
						Namespace:  "default",
						Group:      "apps",
						Kind:       "Deployment",
						SkipReason: "some skip reason",
					},
				},
			},
		},
		"skipped prune resource without reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.PruneAction,
								Name:   "prune-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.PruneType,
					PruneEvent: event.PruneEvent{
						GroupName:  "prune-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.PruneSkipped,
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:      Skip,
						Name:      "foo",
						Namespace: "default",
						Group:     "apps",
						Kind:      "Deployment",
					},
				},
			},
		},
		"skipped delete resource with reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.DeleteAction,
								Name:   "delete-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.DeleteType,
					DeleteEvent: event.DeleteEvent{
						GroupName:  "delete-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.DeleteSkipped,
						Error:      fmt.Errorf("some skip reason"),
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:       Skip,
						Name:       "foo",
						Namespace:  "default",
						Group:      "apps",
						Kind:       "Deployment",
						SkipReason: "some skip reason",
					},
				},
			},
		},
		"skipped delete resource without reason": {
			resources: []*unstructured.Unstructured{
				testutil.Unstructured(t, deploymentYAML),
			},
			clusterResources: []*unstructured.Unstructured{},
			events: []event.Event{
				{
					Type: event.InitType,
					InitEvent: event.InitEvent{
						ActionGroups: event.ActionGroupList{
							{
								Action: event.DeleteAction,
								Name:   "delete-1",
								Identifiers: []object.ObjMetadata{
									testutil.ToIdentifier(t, deploymentYAML),
								},
							},
						},
					},
				},
				{
					Type: event.DeleteType,
					DeleteEvent: event.DeleteEvent{
						GroupName:  "delete-1",
						Identifier: testutil.ToIdentifier(t, deploymentYAML),
						Status:     event.DeleteSkipped,
					},
				},
			},
			expectedPlan: &Plan{
				Actions: []Action{
					{
						Type:      Skip,
						Name:      "foo",
						Namespace: "default",
						Group:     "apps",
						Kind:      "Deployment",
					},
				},
			},
		},
	}

	for tn := range testCases {
		tc := testCases[tn]
		t.Run(tn, func(t *testing.T) {
			ctx := context.Background()

			applier := &FakeApplier{
				events: tc.events,
			}

			fakeResourceFetcher := &FakeResourceFetcher{
				resources: tc.clusterResources,
			}

			plan, err := (&ClusterPlanner{
				applier:         applier,
				resourceFetcher: fakeResourceFetcher,
			}).BuildPlan(ctx, &FakeInventoryInfo{}, []*unstructured.Unstructured{}, Options{})
			require.NoError(t, err)

			diff := cmp.Diff(tc.expectedPlan, plan)
			assert.Empty(t, diff, "plan mismatch (-want +got):\n%s", diff)
		})
	}
}

type FakeApplier struct {
	events []event.Event
}

func (f *FakeApplier) Run(context.Context, inventory.Info, object.UnstructuredSet, apply.ApplierOptions) <-chan event.Event {
	eventChannel := make(chan event.Event)
	go func() {
		defer close(eventChannel)
		for i := range f.events {
			eventChannel <- f.events[i]
		}
	}()
	return eventChannel
}

type FakeResourceFetcher struct {
	resources []*unstructured.Unstructured
}

func (frf *FakeResourceFetcher) FetchResource(_ context.Context, id object.ObjMetadata) (*unstructured.Unstructured, bool, error) {
	for i := range frf.resources {
		r := frf.resources[i]
		rid := object.UnstructuredToObjMetadata(r)
		if rid == id {
			return r, true, nil
		}
	}
	return nil, false, nil
}

type FakeInventoryInfo struct{}

func (fii *FakeInventoryInfo) Namespace() string {
	return ""
}

func (fii *FakeInventoryInfo) Name() string {
	return ""
}

func (fii *FakeInventoryInfo) ID() string {
	return ""
}

func (fii *FakeInventoryInfo) Strategy() inventory.Strategy {
	return inventory.NameStrategy
}

// TestNewClusterPlanner_BackCompatSignaturePreserved pins exported signatures.
func TestNewClusterPlanner_BackCompatSignaturePreserved(t *testing.T) {
	pinSignatures := func(
		_ func(util.Factory) (*ClusterPlanner, error),
		_ func(context.Context, util.Factory) (*ClusterPlanner, error),
	) {
	}
	pinSignatures(NewClusterPlanner, NewClusterPlannerWithContext)
}
