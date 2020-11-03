// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
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

func (f *FakeResourceGroupProvider) ToRESTMapper() (meta.RESTMapper, error) {
	return f.factory.ToRESTMapper()
}

func (f *FakeResourceGroupProvider) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	mapper, err := f.factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	readerOptions := manifestreader.ReaderOptions{
		Mapper:    mapper,
		Namespace: metav1.NamespaceDefault,
	}
	return &ResourceGroupStreamManifestReader{
		streamReader: &manifestreader.StreamManifestReader{
			ReaderName:    "stdin",
			Reader:        reader,
			ReaderOptions: readerOptions,
		},
	}, nil
}
