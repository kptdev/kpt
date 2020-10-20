// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package live

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// InventoryResourceGroup wraps a ResourceGroup resource and implements
// the Inventory interface. This wrapper loads and stores the
// object metadata (inventory) to and from the wrapped ResourceGroup.
type InventoryResourceGroup struct {
	inv      *resource.Info
	objMetas []object.ObjMetadata
}

// WrapInventoryObj takes a passed ResourceGroup (as a resource.Info),
// wraps it with the InventoryResourceGroup and upcasts the wrapper as
// an the Inventory interface.
func WrapInventoryObj(info *resource.Info) inventory.Inventory {
	klog.V(4).Infof("wrapping inventory info")
	return &InventoryResourceGroup{inv: info}
}

// Load is an Inventory interface function returning the set of
// object metadata from the wrapped ResourceGroup, or an error.
func (icm *InventoryResourceGroup) Load() ([]object.ObjMetadata, error) {
	objs := []object.ObjMetadata{}
	if icm.inv == nil {
		return objs, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("loading inventory...")
	inventoryObj, ok := icm.inv.Object.(*unstructured.Unstructured)
	if !ok {
		err := fmt.Errorf("inventory object is not an Unstructured: %#v", inventoryObj)
		return objs, err
	}
	items, exists, err := unstructured.NestedSlice(inventoryObj.Object, "spec", "resources")
	if err != nil {
		err := fmt.Errorf("error retrieving object metadata from inventory object")
		return objs, err
	}
	if !exists {
		klog.V(4).Infof("Inventory (spec.resources) does not exist")
		return objs, nil
	}
	klog.V(4).Infof("loading %d inventory items", len(items))
	for _, itemUncast := range items {
		item := itemUncast.(map[string]interface{})
		namespace, _, err := unstructured.NestedString(item, "namespace")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		name, _, err := unstructured.NestedString(item, "name")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		group, _, err := unstructured.NestedString(item, "group")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		kind, _, err := unstructured.NestedString(item, "kind")
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		groupKind := schema.GroupKind{
			Group: strings.TrimSpace(group),
			Kind:  strings.TrimSpace(kind),
		}
		klog.V(4).Infof("creating obj metadata: %s/%s/%s", namespace, name, groupKind)
		objMeta, err := object.CreateObjMetadata(namespace, name, groupKind)
		if err != nil {
			return []object.ObjMetadata{}, err
		}
		objs = append(objs, objMeta)
	}
	return objs, nil
}

// Store is an Inventory interface function implemented to store
// the object metadata in the wrapped ResourceGroup. Actual storing
// happens in "GetObject".
func (icm *InventoryResourceGroup) Store(objMetas []object.ObjMetadata) error {
	icm.objMetas = objMetas
	return nil
}

// GetObject returns the wrapped object (ResourceGroup) as a resource.Info
// or an error if one occurs.
func (icm *InventoryResourceGroup) GetObject() (*resource.Info, error) {
	if icm.inv == nil {
		return nil, fmt.Errorf("inventory info is nil")
	}
	klog.V(4).Infof("getting inventory resource group")
	// Verify the ResourceGroup is in Unstructured format.
	obj := icm.inv.Object
	if obj == nil {
		return nil, fmt.Errorf("inventory info has nil Object")
	}
	iot, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("inventory ResourceGroup is not in Unstructured format")
	}
	// Create a slice of Resources as empty Interface
	klog.V(4).Infof("Creating list of %d resources", len(icm.objMetas))
	var objs []interface{}
	for _, objMeta := range icm.objMetas {
		klog.V(4).Infof("storing inventory obj refercence: %s/%s", objMeta.Namespace, objMeta.Name)
		objs = append(objs, map[string]interface{}{
			"group":     objMeta.GroupKind.Group,
			"kind":      objMeta.GroupKind.Kind,
			"namespace": objMeta.Namespace,
			"name":      objMeta.Name,
		})
	}
	// Create the inventory object by copying the template.
	invCopy := iot.DeepCopy()
	// Adds the inventory ObjMetadata to the ResourceGroup "spec.resources" section
	klog.V(4).Infof("storing inventory resources")
	err := unstructured.SetNestedSlice(invCopy.UnstructuredContent(),
		objs, "spec", "resources")
	if err != nil {
		return nil, err
	}
	return &resource.Info{
		Client:    icm.inv.Client,
		Mapping:   icm.inv.Mapping,
		Source:    "generated",
		Name:      invCopy.GetName(),
		Namespace: invCopy.GetNamespace(),
		Object:    invCopy,
	}, nil
}
