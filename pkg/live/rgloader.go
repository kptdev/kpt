// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

var _ manifestreader.ManifestLoader = &ResourceGroupManifestLoader{}

// ResourceGroupManifestLoader implements the Provider interface, returning
// ResourceGroup versions of some kpt live apply structures.
type ResourceGroupManifestLoader struct {
	factory util.Factory
}

// NewResourceGroupProvider encapsulates the passed values, and returns a pointer to an ResourceGroupProvider.
func NewResourceGroupManifestLoader(f util.Factory) *ResourceGroupManifestLoader {
	return &ResourceGroupManifestLoader{
		factory: f,
	}
}

// Factory returns the kubectl factory.
func (f *ResourceGroupManifestLoader) InventoryInfo(objs []*unstructured.Unstructured) (inventory.InventoryInfo, []*unstructured.Unstructured, error) {
	objs, invObj := findResourceGroupInv(objs)
	if invObj == nil {
		return nil, nil, fmt.Errorf("unable to find the ResourceGroup inventory info")
	}
	return &InventoryResourceGroup{inv: invObj}, objs, nil
}

// ManifestReader returns the ResourceGroup inventory object version of
// the ManifestReader.
func (f *ResourceGroupManifestLoader) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	// Validate parameters.
	if reader == nil && len(args) == 0 {
		return nil, fmt.Errorf("unable to build ManifestReader without both reader or args")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("expected one directory argument allowed; got (%s)", args)
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
