// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

var _ provider.Provider = &DualDelegatingProvider{}

// DualDelegatingProvider encapsulates another Provider to which it
// delegates most functions, enabling a Provider which will return
// values based on the inventory object found from ManifestReader()
// call.
type DualDelegatingProvider struct {
	// ResourceGroupProvider is the delegate.
	rgProvider provider.Provider
	// Default inventory function is for ConfigMap.
	wrapInv inventory.InventoryFactoryFunc
	// Boolean on whether we've calculated the inventory object type.
	calcInventory bool
}

// NewDualDelagatingProvider returns a pointer to the DualDelegatingProvider,
// setting default values.
func NewDualDelegatingProvider(f util.Factory) *DualDelegatingProvider {
	return &DualDelegatingProvider{
		rgProvider:    NewResourceGroupProvider(f),
		wrapInv:       inventory.WrapInventoryObj,
		calcInventory: false,
	}
}

// Factory returns the delegate factory.
func (cp *DualDelegatingProvider) Factory() util.Factory {
	return cp.rgProvider.Factory()
}

// InventoryClient returns an InventoryClient that is created from the
// stored/calculated InventoryFactoryFunction. This must be called
// after ManifestReader().
func (cp *DualDelegatingProvider) InventoryClient() (inventory.InventoryClient, error) {
	if !cp.calcInventory {
		return nil, fmt.Errorf("must be called after ManifestReader()")
	}
	return inventory.NewInventoryClient(cp.Factory(), cp.wrapInv)
}

// ToRESTMapper returns the value from the delegate provider; or an error.
func (cp *DualDelegatingProvider) ToRESTMapper() (meta.RESTMapper, error) {
	return cp.Factory().ToRESTMapper()
}

// ManifestReader retrieves the ManifestReader from the delegate ResourceGroup
// Provider, then calls Read() for this ManifestReader to retrieve the objects
// and to calculate the type of Inventory object is present. Returns a
// CachedManifestReader with the read objects, or an error. Can return a
// NoInventoryError or MultipleInventoryError.
func (cp *DualDelegatingProvider) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	r, err := cp.rgProvider.ManifestReader(reader, args)
	if err != nil {
		return nil, err
	}
	objs, err := r.Read()
	if err != nil {
		return nil, err
	}
	klog.V(4).Infof("ManifestReader read %d objects", len(objs))
	rgInv := findResourceGroupInv(objs)
	// A ResourceGroup inventory object means we need an InventoryFactoryFunc
	// which works for ResourceGroup (instead of ConfigMap, which is default).
	if rgInv != nil {
		cp.wrapInv = WrapInventoryObj
	}
	cmInv := findConfigMapInv(objs)
	if rgInv == nil && cmInv == nil {
		return nil, inventory.NoInventoryObjError{}
	}
	if rgInv != nil && cmInv != nil {
		return nil, inventory.MultipleInventoryObjError{
			InventoryObjectTemplates: []*unstructured.Unstructured{rgInv, cmInv},
		}
	}
	cp.calcInventory = true
	return &CachedManifestReader{objs: objs}, nil
}

// CachedManifestReader implements ManifestReader, storing objects to return.
type CachedManifestReader struct {
	objs []*unstructured.Unstructured
}

// Read simply returns the stored objects.
func (r *CachedManifestReader) Read() ([]*unstructured.Unstructured, error) {
	return r.objs, nil
}

// findResourceGroupInv returns the pointer to the ResourceGroup inventory object,
// or nil if it does not exist.
func findResourceGroupInv(objs []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, obj := range objs {
		if inventory.IsInventoryObject(obj) {
			if obj.GetKind() == "ResourceGroup" {
				return obj
			}
		}
	}
	return nil
}

// findConfigMapInv returns the pointer to the ConfigMap inventory object,
// or nil if it does not exist.
func findConfigMapInv(objs []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, obj := range objs {
		if inventory.IsInventoryObject(obj) {
			if obj.GetKind() == "ConfigMap" {
				return obj
			}
		}
	}
	return nil
}
