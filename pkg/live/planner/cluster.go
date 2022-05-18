// Copyright 2022 Google LLC
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
	"reflect"

	planv1alpha1 "github.com/GoogleContainerTools/kpt/pkg/api/plan/v1alpha1"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func NewClusterPlanner(f util.Factory) (*ClusterPlanner, error) {
	fetcher, err := NewResourceFetcher(f)
	if err != nil {
		return nil, err
	}

	invClient, err := inventory.NewClient(f, live.WrapInventoryObj, live.InvToUnstructuredFunc, inventory.StatusPolicyNone)
	if err != nil {
		return nil, err
	}

	applier, err := apply.NewApplierBuilder().
		WithFactory(f).
		WithInventoryClient(invClient).
		Build()
	if err != nil {
		return nil, err
	}

	return &ClusterPlanner{
		applier:         applier,
		resourceFetcher: fetcher,
	}, nil
}

type Applier interface {
	Run(ctx context.Context, invInfo inventory.Info, objects object.UnstructuredSet, options apply.ApplierOptions) <-chan event.Event
}

type ResourceFetcher interface {
	FetchResource(ctx context.Context, id object.ObjMetadata) (*unstructured.Unstructured, bool, error)
}

type ClusterPlanner struct {
	applier         Applier
	resourceFetcher ResourceFetcher
}

type Options struct {
	ServerSideOptions common.ServerSideOptions
}

func (r *ClusterPlanner) BuildPlan(ctx context.Context, inv inventory.Info, objects []*unstructured.Unstructured, o Options) (*planv1alpha1.Plan, error) {
	actions, err := r.dryRunForPlan(ctx, inv, objects, o)
	if err != nil {
		return nil, err
	}
	return &planv1alpha1.Plan{
		ResourceMeta: planv1alpha1.ResourceMeta,
		Spec: planv1alpha1.PlanSpec{
			Actions: actions,
		},
	}, nil
}

func (r *ClusterPlanner) dryRunForPlan(
	ctx context.Context,
	inv inventory.Info,
	objects []*unstructured.Unstructured,
	o Options,
) ([]planv1alpha1.Action, error) {
	eventCh := r.applier.Run(ctx, inv, objects, apply.ApplierOptions{
		DryRunStrategy:    common.DryRunServer,
		ServerSideOptions: o.ServerSideOptions,
	})

	var actions []planv1alpha1.Action
	var err error
	for e := range eventCh {
		if e.Type == event.InitType {
			// This event includes all resources that will be applied, pruned or deleted, so
			// we make sure we fetch all the resources from the cluster.
			// TODO: See if we can update the actuation library to provide the pre-actuation
			// versions of the resources as part of the regular run. This solution is not great
			// as fetching all resources will take time.
			a, err := r.fetchResources(ctx, e)
			if err != nil {
				return nil, err
			}
			actions = a
		}
		if e.Type == event.ErrorType {
			// Update the err variable here, but wait for the channel to close
			// before we return from the function.
			// Since ErrorEvents are considered fatal, there should only be sent
			// and it will be followed by the channel being closed.
			err = e.ErrorEvent.Err
		}
		// For the Apply, Prune and Delete event types, we just capture the result
		// of the dry-run operation for the specific resource.
		switch e.Type {
		case event.ApplyType:
			id := e.ApplyEvent.Identifier
			index := indexForIdentifier(id, actions)
			a := actions[index]
			actions[index] = handleApplyEvent(e, a)
		case event.PruneType:
			id := e.PruneEvent.Identifier
			index := indexForIdentifier(id, actions)
			a := actions[index]
			actions[index] = handlePruneEvent(e, a)
		// Prune and Delete are essentially the same thing, but the actuation
		// library return Prune events when resources are deleted by omission
		// during apply, and Delete events from the destroyer. Supporting both
		// here for completeness.
		case event.DeleteType:
			id := e.DeleteEvent.Identifier
			index := indexForIdentifier(id, actions)
			a := actions[index]
			actions[index] = handleDeleteEvent(e, a)
		}
	}
	return actions, err
}

func handleApplyEvent(e event.Event, a planv1alpha1.Action) planv1alpha1.Action {
	if e.ApplyEvent.Error != nil {
		a.Type = planv1alpha1.Error
		a.Error = e.ApplyEvent.Error.Error()
	} else {
		switch e.ApplyEvent.Status {
		case event.ApplySkipped:
			a.Type = planv1alpha1.Skip
		case event.ApplySuccessful:
			a.After = e.ApplyEvent.Resource
			if a.Before != nil {
				// TODO: Unclear if we should diff the full resources here. It doesn't work
				// well with client-side apply as the managedFields property shows up as
				// changes. It also means there is a race with controllers that might change
				// the status of resources.
				if reflect.DeepEqual(a.Before, a.After) {
					a.Type = planv1alpha1.Unchanged
				} else {
					a.Type = planv1alpha1.Update
				}
			} else {
				a.Type = planv1alpha1.Create
			}
		}
	}
	return a
}

func handlePruneEvent(e event.Event, a planv1alpha1.Action) planv1alpha1.Action {
	if e.PruneEvent.Error != nil {
		a.Type = planv1alpha1.Error
		a.Error = e.PruneEvent.Error.Error()
	} else {
		switch e.PruneEvent.Status {
		case event.PruneSuccessful:
			a.Type = planv1alpha1.Delete
		// Lifecycle directives can cause resources to remain in the
		// live state even if they would normally be pruned.
		// TODO: Handle reason for skipped resources that has recently
		// been added to the actuation library.
		case event.PruneSkipped:
			a.Type = planv1alpha1.Skip
		}
	}
	return a
}

func handleDeleteEvent(e event.Event, a planv1alpha1.Action) planv1alpha1.Action {
	if e.DeleteEvent.Error != nil {
		a.Type = planv1alpha1.Error
		a.Error = e.DeleteEvent.Error.Error()
	} else {
		switch e.DeleteEvent.Status {
		case event.DeleteSuccessful:
			a.Type = planv1alpha1.Delete
		case event.DeleteSkipped:
			a.Type = planv1alpha1.Skip
		}
	}
	return a
}

func (r *ClusterPlanner) fetchResources(ctx context.Context, e event.Event) ([]planv1alpha1.Action, error) {
	var actions []planv1alpha1.Action
	for _, ag := range e.InitEvent.ActionGroups {
		// We only care about the Apply, Prune and Delete actions.
		if !(ag.Action == event.ApplyAction || ag.Action == event.PruneAction || ag.Action == event.DeleteAction) {
			continue
		}
		for _, id := range ag.Identifiers {
			u, _, err := r.resourceFetcher.FetchResource(ctx, id)
			// If the type doesn't exist in the cluster, then the resource itself doesn't exist.
			if err != nil && !meta.IsNoMatchError(err) {
				return nil, err
			}
			actions = append(actions, planv1alpha1.Action{
				Group:     id.GroupKind.Group,
				Kind:      id.GroupKind.Kind,
				Name:      id.Name,
				Namespace: id.Namespace,
				Before:    u,
			})
		}
	}
	return actions, nil
}

type resourceFetcher struct {
	dynamicClient dynamic.Interface
	mapper        meta.RESTMapper
}

func NewResourceFetcher(f util.Factory) (ResourceFetcher, error) {
	dc, err := f.DynamicClient()
	if err != nil {
		return nil, err
	}

	mapper, err := f.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	return &resourceFetcher{
		dynamicClient: dc,
		mapper:        mapper,
	}, nil
}

func (rf *resourceFetcher) FetchResource(ctx context.Context, id object.ObjMetadata) (*unstructured.Unstructured, bool, error) {
	mapping, err := rf.mapper.RESTMapping(id.GroupKind)
	if err != nil {
		return nil, false, err
	}
	var r dynamic.ResourceInterface
	if mapping.Scope == meta.RESTScopeRoot {
		r = rf.dynamicClient.Resource(mapping.Resource)
	} else {
		r = rf.dynamicClient.Resource(mapping.Resource).Namespace(id.Namespace)
	}
	u, err := r.Get(ctx, id.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, false, err
	}

	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	return u, true, nil
}

func indexForIdentifier(id object.ObjMetadata, actions []planv1alpha1.Action) int {
	for i := range actions {
		a := actions[i]
		if a.Group == id.GroupKind.Group &&
			a.Kind == id.GroupKind.Kind &&
			a.Name == id.Name &&
			a.Namespace == id.Namespace {
			return i
		}
	}
	panic(fmt.Errorf("unknown identifier %s", id.String()))
}
