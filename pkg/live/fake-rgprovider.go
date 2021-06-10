// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

type FakeResourceGroupProvider struct {
	factory   util.Factory
	InvClient *inventory.FakeInventoryClient
}

var _ provider.Provider = &FakeResourceGroupProvider{}

func NewFakeResourceGroupProvider(f util.Factory, objs []object.ObjMetadata) *FakeResourceGroupProvider {
	return &FakeResourceGroupProvider{
		factory:   f,
		InvClient: inventory.NewFakeInventoryClient(objs),
	}
}

func (f *FakeResourceGroupProvider) Factory() util.Factory {
	return f.factory
}

func (f *FakeResourceGroupProvider) InventoryClient() (inventory.InventoryClient, error) {
	return f.InvClient, nil
}
