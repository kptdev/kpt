// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
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

// ToRESTMapper returns the RESTMapper or an erro if one occurred.
func (f *ResourceGroupProvider) ToRESTMapper() (meta.RESTMapper, error) {
	return f.factory.ToRESTMapper()
}

// ManifestReader returns the ResourceGroup inventory object version of
// the ManifestReader.
func (f *ResourceGroupProvider) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	// Validate parameters.
	if reader == nil && len(args) == 0 {
		return nil, fmt.Errorf("unable to build ManifestReader without both reader or args")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("expected one directory argument allowed; got %q", args)
	}
	// Create ReaderOptions for subsequent ManifestReader.
	namespace, enforceNamespace, err := f.factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	mapper, err := f.factory.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	readerOptions := manifestreader.ReaderOptions{
		Mapper:           mapper,
		Namespace:        namespace,
		EnforceNamespace: enforceNamespace,
	}
	// No arguments means stream (using reader), while one argument
	// means path manifest reader.
	var rgReader manifestreader.ManifestReader
	if len(args) == 0 {
		rgReader = &ResourceGroupStreamManifestReader{
			streamReader: &manifestreader.StreamManifestReader{
				ReaderName:    "stdin",
				Reader:        reader,
				ReaderOptions: readerOptions,
			},
		}
	} else {
		rgReader = &ResourceGroupPathManifestReader{
			pathReader: &manifestreader.PathManifestReader{
				Path:          args[0],
				ReaderOptions: readerOptions,
			},
		}
	}
	return rgReader, nil
}
