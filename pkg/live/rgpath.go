// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

// ResourceGroupPathManifestReader encapsulates the default path
// manifest reader.
type ResourceGroupPathManifestReader struct {
	pathReader *manifestreader.PathManifestReader
}

// Read reads the manifests and returns them as Info objects.
// Generates and adds a ResourceGroup inventory object from
// Kptfile data. If unable to generate the ResourceGroup inventory
// object from the Kptfile, it is NOT an error.
func (p *ResourceGroupPathManifestReader) Read() ([]*unstructured.Unstructured, error) {
	// Using the default path reader to generate the objects.
	objs, err := p.pathReader.Read()
	if err != nil {
		return []*unstructured.Unstructured{}, err
	}
	klog.V(4).Infof("path Read() %d resources", len(objs))
	// Read the Kptfile in the top directory to get the inventory
	// parameters to create the ResourceGroup inventory object.
	kf, err := kptfileutil.ReadFile(p.pathReader.Path)
	if err != nil {
		klog.V(4).Infof("unable to parse Kptfile for ResourceGroup inventory: %s", err)
		return objs, nil
	}
	inv := kf.Inventory
	invObj, err := generateInventoryObj(inv)
	if err == nil {
		klog.V(4).Infof("from Kptfile generating ResourceGroup inventory object %s/%s/%s",
			inv.Namespace, inv.Name, inv.InventoryID)
		objs = append(objs, invObj)
	} else {
		klog.V(4).Infof("unable to generate ResourceGroup inventory: %s", err)
	}
	return objs, nil
}

// generateInventoryObj returns the ResourceGroupInventory object using the
// passed information.
func generateInventoryObj(inv *kptfile.Inventory) (*unstructured.Unstructured, error) {
	// First, ensure the Kptfile inventory section is valid.
	if isValid, err := kptfileutil.ValidateInventory(inv); !isValid {
		return nil, err
	}
	// Create and return ResourceGroup custom resource as inventory object.
	var inventoryObj = ResourceGroupUnstructured(inv.Name, inv.Namespace, inv.InventoryID)
	labels := inv.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[common.InventoryLabel] = inv.InventoryID
	inventoryObj.SetLabels(labels)
	inventoryObj.SetAnnotations(inv.Annotations)
	return inventoryObj, nil
}

func ResourceGroupUnstructured(name, namespace, id string) *unstructured.Unstructured {
	groupVersion := fmt.Sprintf("%s/%s", ResourceGroupGVK.Group, ResourceGroupGVK.Version)
	inventoryObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": groupVersion,
			"kind":       ResourceGroupGVK.Kind,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
				"labels": map[string]interface{}{
					common.InventoryLabel: id,
				},
			},
			"spec": map[string]interface{}{
				"resources": []interface{}{},
			},
		},
	}
	return inventoryObj
}
