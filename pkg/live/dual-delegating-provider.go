// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
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
}

// NewDualDelagatingProvider returns a pointer to the DualDelegatingProvider,
// setting default values.
func NewDualDelegatingProvider(f util.Factory) *DualDelegatingProvider {
	return &DualDelegatingProvider{
		rgProvider: NewResourceGroupProvider(f),
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
	return inventory.NewInventoryClient(cp.Factory(),
		inventoryWrapperFunc,
		invToUnstructuredFunc)
}

func inventoryWrapperFunc(obj *unstructured.Unstructured) inventory.Inventory {
	switch obj.GetKind() {
	case "ResourceGroup":
		return &InventoryResourceGroup{inv: obj}
	case "ConfigMap":
		return inventory.WrapInventoryObj(obj)
	default:
		return nil
	}
}

func invToUnstructuredFunc(inv inventory.InventoryInfo) *unstructured.Unstructured {
	switch invInfo := inv.(type) {
	case *InventoryResourceGroup:
		return invInfo.inv
	case *inventory.InventoryConfigMap:
		return invInfo.UnstructuredInventory()
	default:
		return nil
	}
}
