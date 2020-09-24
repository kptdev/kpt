// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

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
	return inventory.NewInventoryClient(f.factory, WrapInventoryObj)
}

// ToRESTMapper returns the RESTMapper or an erro if one occurred.
func (f *ResourceGroupProvider) ToRESTMapper() (meta.RESTMapper, error) {
	return f.factory.ToRESTMapper()
}

// ManifestReader returns the ResourceGroup inventory object version of
// the ManifestReader.
func (f *ResourceGroupProvider) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	namespace, enforceNamespace, err := f.factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}

	readerOptions := manifestreader.ReaderOptions{
		Factory:          f.factory,
		Namespace:        namespace,
		EnforceNamespace: enforceNamespace,
	}

	// TODO(seans3): Add Kptfile/ResourceGroup version of StreamManifestReader,
	// when there are no args.

	pathReader := &manifestreader.PathManifestReader{
		Path:          args[0],
		ReaderOptions: readerOptions,
	}

	mReader := &ResourceGroupPathManifestReader{
		pathReader: pathReader,
	}
	return mReader, nil
}
