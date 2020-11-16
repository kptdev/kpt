// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

var _ manifestreader.ManifestLoader = &DualDelegatingManifestReader{}

// DualDelegatingManifestReader read manifests that either uses ConfigMap
// or the ResourceGroup as the inventory object.
type DualDelegatingManifestReader struct {
	factory util.Factory
}

var _ manifestreader.ManifestLoader = &DualDelegatingManifestReader{}

func NewDualDelegatingManifestReader(f util.Factory) *DualDelegatingManifestReader {
	return &DualDelegatingManifestReader{factory: f}
}

// ManifestReader retrieves the ManifestReader from the delegate ResourceGroup
// Provider, then calls Read() for this ManifestReader to retrieve the objects
// and to calculate the type of Inventory object is present. Returns a
// CachedManifestReader with the read objects, or an error.
func (cp *DualDelegatingManifestReader) ManifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	r, err := cp.manifestReader(reader, args)
	if err != nil {
		return nil, err
	}
	objs, err := r.Read()
	if err != nil {
		return nil, err
	}
	klog.V(4).Infof("ManifestReader read %d objects", len(objs))
	return &CachedManifestReader{objs: objs}, nil
}

func (cp *DualDelegatingManifestReader) manifestReader(reader io.Reader, args []string) (manifestreader.ManifestReader, error) {
	// Validate parameters.
	if reader == nil && len(args) == 0 {
		return nil, fmt.Errorf("unable to build ManifestReader without both reader or args")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("expected one directory argument allowed; got (%s)", args)
	}
	// Create ReaderOptions for subsequent ManifestReader.
	namespace, enforceNamespace, err := cp.factory.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, err
	}
	mapper, err := cp.factory.ToRESTMapper()
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

// InventoryInfo returns the InventoryInfo from a list of Unstructured objects.
// It can return a NoInventoryError or MultipleInventoryError.
func (cp *DualDelegatingManifestReader) InventoryInfo(objs []*unstructured.Unstructured) (inventory.InventoryInfo, []*unstructured.Unstructured, error) {
	objs, rgInv := findResourceGroupInv(objs)
	var inv inventory.InventoryInfo
	// A ResourceGroup inventory object means we need an InventoryFactoryFunc
	// which works for ResourceGroup (instead of ConfigMap, which is default).
	if rgInv != nil {
		inv = &InventoryResourceGroup{inv: rgInv}
	}
	objs, cmInv := findConfigMapInv(objs)
	if rgInv == nil && cmInv == nil {
		return nil, objs, inventory.NoInventoryObjError{}
	}
	if rgInv != nil && cmInv != nil {
		return nil, objs, MultipleInventoryObjError{
			InvObjs: []*unstructured.Unstructured{rgInv, cmInv},
		}
	}
	if cmInv != nil {
		inv = inventory.WrapInventoryInfoObj(cmInv)
	}
	return inv, objs, nil
}

// MultipleInventoryObjError is thrown when more than one inventory
// objects is detected.
type MultipleInventoryObjError struct {
	InvObjs []*unstructured.Unstructured
}

const multipleInventoryErrorStr = `Detected ResourceGroup (Kptfile) and deprecated ConfigMap (inventory-template.yaml)
inventory objects. Please run "kpt live migrate" to complete upgrade to
ResourceGroup inventory object.
`

func (g MultipleInventoryObjError) Error() string {
	return multipleInventoryErrorStr
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
func findResourceGroupInv(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, *unstructured.Unstructured) {
	var fileteredObjs []*unstructured.Unstructured
	var inventoryObj *unstructured.Unstructured
	for _, obj := range objs {
		if inventory.IsInventoryObject(obj) {
			if obj.GetKind() == "ResourceGroup" {
				inventoryObj = obj
				continue
			}
		}
		fileteredObjs = append(fileteredObjs, obj)
	}
	return fileteredObjs, inventoryObj
}

// findConfigMapInv returns the pointer to the ConfigMap inventory object,
// or nil if it does not exist.
func findConfigMapInv(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, *unstructured.Unstructured) {
	var fileteredObjs []*unstructured.Unstructured
	var inventoryObj *unstructured.Unstructured
	for _, obj := range objs {
		if inventory.IsInventoryObject(obj) {
			if obj.GetKind() == "ConfigMap" {
				inventoryObj = obj
				continue
			}
		}
		fileteredObjs = append(fileteredObjs, obj)
	}
	return fileteredObjs, inventoryObj
}
