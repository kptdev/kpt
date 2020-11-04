// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"strings"

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
	// Read the Kptfile in the top directory to get the inventory
	// parameters to create the ResourceGroup inventory object.
	kf, err := kptfileutil.ReadFile(p.pathReader.Path)
	if err == nil {
		inv := kf.Inventory
		klog.V(4).Infof("from Kptfile generating ResourceGroup inventory object %s/%s/%s",
			inv.Namespace, inv.Name, inv.InventoryID)
		invObj, err := generateInventoryObj(inv)
		if err == nil {
			objs = append(objs, invObj)
		} else {
			klog.V(4).Infof("unable to generate ResourceGroup inventory: %s", err)
		}
	} else {
		klog.V(4).Infof("unable to parse Kpfile for ResourceGroup inventory: %s", err)
	}
	klog.V(4).Infof("path Read() generated %d resources", len(objs))
	return objs, nil
}

// generateInventoryObj returns the ResourceGroupInventory object using the
// passed information.
func generateInventoryObj(inv kptfile.Inventory) (*unstructured.Unstructured, error) {
	// Validate the parameters
	name := strings.TrimSpace(inv.Name)
	if name == "" {
		return nil, fmt.Errorf("kptfile inventory empty name")
	}
	namespace := strings.TrimSpace(inv.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("kptfile inventory empty namespace")
	}
	id := strings.TrimSpace(inv.InventoryID)
	if id == "" {
		return nil, fmt.Errorf("kptfile inventory missing inventoryID")
	}
	// Create and return ResourceGroup custom resource as inventory object.
	groupVersion := fmt.Sprintf("%s/%s", ResourceGroupGVK.Group, ResourceGroupGVK.Version)
	var inventoryObj = &unstructured.Unstructured{
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
	labels := inv.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[common.InventoryLabel] = id
	inventoryObj.SetLabels(labels)
	inventoryObj.SetAnnotations(inv.Annotations)
	return inventoryObj, nil
}
