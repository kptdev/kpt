// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package nested

import (
	"context"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"github.com/go-errors/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/apply"
	"sigs.k8s.io/cli-utils/pkg/apply/event"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

type applier struct {
	provider  provider.Provider
	invclient inventory.InventoryClient
	applier   *apply.Applier
	destroyer *apply.Destroyer
}

func NewApplier(p provider.Provider) (*applier, error) {
	invClient, err := p.InventoryClient()
	if err != nil {
		return nil, err
	}
	ap := apply.NewApplier(p)
	err = ap.Initialize()
	if err != nil {
		return nil, err
	}
	ds := apply.NewDestroyer(p)
	err = ds.Initialize()
	if err != nil {
		return nil, err
	}

	return &applier{
		provider:  p,
		invclient: invClient,
		applier:   ap,
		destroyer: ds,
	}, nil
}

func (a *applier) Apply(ctx context.Context, ninv *NestedInventory, option apply.Options) <-chan event.Event {
	eventChannel := make(chan event.Event)
	go func() {
		defer close(eventChannel)

		// first step: apply inventory resourcegroups with updating the subgroup inventory list as the union
		klog.V(6).Infof("starting union all the inventory lists under the root: %v", ninv.Resourcegroup)
		if err := a.unionInventoryList(ctx, ninv); err != nil {
			klog.V(6).Infof("received an error for union all the inventory lists: %v", err)
			eventChannel <- event.Event{
				Type: event.ErrorType,
				ErrorEvent: event.ErrorEvent{
					Err: errors.WrapPrefix(err, "failed to union inventory lists", 1),
				},
			}
			return
		}
		klog.V(6).Infof("finished union all the inventory lists under the root: %s", ninv.Resourcegroup.ID())

		// second step: prune the inventory objects that have been removed in the new apply
		klog.V(6).Infof("starting pruning the obsolete inventory objects under the root: %s", ninv.Resourcegroup.ID())
		a.pruneSubpackage(ctx, ninv)
		klog.V(6).Infof("finished pruning the obsolete inventory objects under the root: %s", ninv.Resourcegroup.ID())

		// third step: apply each inventory object by triggering the cli-utils applier
		klog.V(6).Infof("starting applying individual package under the root: %s", ninv.Resourcegroup.ID())
		a.applySubpackage(ctx, eventChannel, ninv)
		klog.V(6).Infof("finished applying individual package under the root: %s", ninv.Resourcegroup.ID())

		// four step: update the final inventory resourcegroups for the subgroup inventory list.
		klog.V(6).Infof("starting updating final inventory lists under the root: %s", ninv.Resourcegroup.ID())
		a.finalUpdateInventoryList(ctx, eventChannel, ninv)
		klog.V(6).Infof("finished updating final inventory lists under the root: %s", ninv.Resourcegroup.ID())
	}()

	return eventChannel
}

func (a *applier) unionInventoryList(ctx context.Context, ninv *NestedInventory) error {
	if ninv == nil {
		return nil
	}

	// update live.AllGroups so that the resources being moved between packages
	// can be handled correctly.
	live.AllGroups = append(live.AllGroups, ninv.Resourcegroup.ID())

	// update the top inventory object
	clusterObject, err := a.invclient.GetClusterInventoryInfo(ninv.Resourcegroup)
	if err != nil && apierrors.IsNotFound(err) || clusterObject == nil {
		klog.V(6).Infof("Inventory object for inventory %s doesn't exist in the cluster", ninv.Resourcegroup.ID())
		obj, err := ninv.Resourcegroup.StoreSubgroups(ninv.newChildren)
		if err != nil {
			klog.V(6).Infof("failed to update the subgroup list for inventory %s", ninv.Resourcegroup.ID())
			return err
		}
		klog.V(6).Infof("Updating the subgroup list for inventory %s", ninv.Resourcegroup.ID())
		err = a.invclient.ApplyInventoryObj(obj)
		if err != nil {
			klog.V(6).Infof("failed to apply the updated inventory object %s", ninv.Resourcegroup.ID())
			return err
		}
		klog.V(6).Infof("Applied the updated inventory object %s", ninv.Resourcegroup.ID())
	} else {
		klog.V(6).Infof("Inventory object for inventory %s exists in the cluster", ninv.Resourcegroup.ID())
		curr := live.WrapInventoryResourceGroup(clusterObject)
		children, err := curr.LoadSubgroups()
		if err != nil {
			return err
		}
		klog.V(6).Infof("Loaded the old children from the cluster object for %s", ninv.Resourcegroup.ID())
		ninv.oldChildren = children
		m := map[object.ObjMetadata]bool{}
		for _, ch := range children {
			m[ch] = true
		}
		for _, ch := range ninv.newChildren {
			m[ch] = true
		}
		children = []object.ObjMetadata{}
		for ch := range m {
			children = append(children, ch)
		}

		klog.V(6).Infof("Updating the subgroup list for inventory %s", ninv.Resourcegroup.ID())
		obj, err := curr.StoreSubgroups(children)
		if err != nil {
			klog.V(6).Infof("failed to update the subgroup list for inventory %s", ninv.Resourcegroup.ID())
			return err
		}
		klog.V(6).Infof("Merged the old children and new children for %s", ninv.Resourcegroup.ID())
		err = a.invclient.ApplyInventoryObj(obj)
		if err != nil {
			klog.V(6).Infof("failed to apply the updated inventory object %s", ninv.Resourcegroup.ID())
			return err
		}
		klog.V(6).Infof("Applied the updated inventory object %s", ninv.Resourcegroup.ID())
	}
	// update the child inventory objects
	for _, ch := range ninv.Children {
		klog.V(6).Infof("union the inventory subgroup for child inventory object %s", ch.Resourcegroup.ID())
		if err := a.unionInventoryList(ctx, ch); err != nil {
			return err
		}
	}
	return nil
}

func (a *applier) applySubpackage(ctx context.Context, eventChannel chan event.Event, ninv *NestedInventory) {
	if ninv == nil {
		return
	}

	events := a.applier.Run(ctx, ninv.Resourcegroup, ninv.Resources, apply.Options{
		ServerSideOptions: common.ServerSideOptions{
			ServerSideApply: false,
		},
		DryRunStrategy:  common.DryRunNone,
		InventoryPolicy: inventory.AdoptIfNoInventory,
	})
	for e := range events {
		eventChannel <- e
	}
	for _, ch := range ninv.Children {
		a.applySubpackage(ctx, eventChannel, ch)
	}
}

func (a *applier) pruneSubpackage(ctx context.Context, ninv *NestedInventory) {
	if ninv == nil {
		return
	}
	for _, ch := range ninv.Children {
		a.pruneSubpackage(ctx, ch)
	}

	m := map[object.ObjMetadata]bool{}
	for _, ch := range ninv.newChildren {
		m[ch] = true
	}

	var failedToDestroy []object.ObjMetadata
	for _, ch := range ninv.oldChildren {
		if !m[ch] {
			client, err := namespacedClient(a.provider, ch)
			obj, err := client.Get(ctx, ch.Name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				failedToDestroy = append(failedToDestroy, ch)
				continue
			}
			events := a.destroyer.Run(live.WrapInventoryResourceGroup(obj),
				&apply.DestroyerOption{InventoryPolicy: inventory.InventoryPolicyMustMatch})
			for e := range events {
				if e.Type == event.ErrorType || e.DeleteEvent.Error != nil {
					failedToDestroy = append(failedToDestroy, ch)
					break
				}
			}
		}
	}

	ninv.newChildren = append(ninv.newChildren, failedToDestroy...)
}

func (a *applier) finalUpdateInventoryList(ctx context.Context, eventChannel chan event.Event, ninv *NestedInventory) {
	if ninv == nil {
		return
	}

	// update the child inventory objects
	for _, ch := range ninv.Children {
		a.finalUpdateInventoryList(ctx, eventChannel, ch)
	}
	// update the top inventory object
	objs, err := a.invclient.GetClusterObjs(ninv.Resourcegroup)
	if err != nil {
		eventChannel <- event.Event{
			Type: event.ErrorType,
			ErrorEvent: event.ErrorEvent{
				Err: errors.WrapPrefix(err, "error getting the resources", 1),
			},
		}
	}
	err = ninv.Resourcegroup.Store(objs)
	if err != nil {
		eventChannel <- event.Event{
			Type: event.ErrorType,
			ErrorEvent: event.ErrorEvent{
				Err: errors.WrapPrefix(err, "error saving the resources", 1),
			},
		}
	}

	obj, err := ninv.Resourcegroup.StoreSubgroups(ninv.newChildren)
	if err != nil {
		eventChannel <- event.Event{
			Type: event.ErrorType,
			ErrorEvent: event.ErrorEvent{
				Err: errors.WrapPrefix(err, "error storing the sub groups", 1),
			},
		}
	}

	if err = a.invclient.ApplyInventoryObj(obj); err != nil {
		eventChannel <- event.Event{
			Type: event.ErrorType,
			ErrorEvent: event.ErrorEvent{
				Err: errors.WrapPrefix(err, "error final updating the sub groups", 1),
			},
		}
	}
}

func namespacedClient(p provider.Provider, obj object.ObjMetadata) (dynamic.ResourceInterface, error) {
	mapper, err := p.Factory().ToRESTMapper()
	if err != nil {
		return nil, err
	}
	mapping, err := mapper.RESTMapping(obj.GroupKind)
	if err != nil {
		return nil, err
	}
	dy, err := p.Factory().DynamicClient()
	if err != nil {
		return nil, err
	}
	return dy.Resource(mapping.Resource).Namespace(obj.Namespace), nil
}
