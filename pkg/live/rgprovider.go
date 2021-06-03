// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

var _ provider.Provider = &ResourceGroupProvider{}

// ResourceGroupProvider implements the Provider interface, returning
// ResourceGroup versions of some kpt live apply structures.
type ResourceGroupProvider struct {
	factory util.Factory
}

// NewResourceGroupProvider encapsulates the passed values, and returns a pointer to an ResourceGroupProvider.
func NewResourceGroupProvider(f util.Factory) *ResourceGroupProvider {
	return &ResourceGroupProvider{
		factory: f,
	}
}

// Factory returns the kubectl factory.
func (f *ResourceGroupProvider) Factory() util.Factory {
	return f.factory
}

// InventoryClient returns the InventoryClient created using the
// ResourceGroup inventory object wrapper function.
func (f *ResourceGroupProvider) InventoryClient() (inventory.InventoryClient, error) {
	return inventory.NewInventoryClient(f.factory, WrapInventoryObj, InvToUnstructuredFunc)
}
