// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/resource"
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
// Kptfile data.
func (p *ResourceGroupPathManifestReader) Read() ([]*resource.Info, error) {
	// Using the default path reader to generate the objects.
	infos, err := p.pathReader.Read()
	if err != nil {
		return []*resource.Info{}, err
	}
	// Read the Kptfile in the top directory to get the inventory
	// parameters to create the ResourceGroup inventory object.
	kf, err := kptfileutil.ReadFile(p.pathReader.Path)
	if err != nil {
		return []*resource.Info{}, err
	}
	inv := kf.Inventory
	klog.V(4).Infof("generating ResourceGroup inventory object %s/%s/%s", inv.Namespace, inv.Name, inv.InventoryID)
	invInfo, err := generateInventoryObj(inv.Name, inv.Namespace, inv.InventoryID)
	if err != nil {
		return []*resource.Info{}, err
	}
	infos = append(infos, invInfo)
	return infos, nil
}

// generateInventoryObj returns the ResourceGroupInventory object using the
// passed information.
func generateInventoryObj(name string, namespace string, id string) (*resource.Info, error) {
	// Validate the parameters
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("kptfile inventory empty name")
	}
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("kptfile inventory empty namespace")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("kptfile inventory missing inventoryID")
	}
	// Create and return ResourceGroup custom resource as inventory object.
	var inventoryObj = unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kpt.dev/v1alpha1",
			"kind":       "ResourceGroup",
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
	var invInfo = &resource.Info{
		Namespace: namespace,
		Name:      name,
		Object:    &inventoryObj,
	}
	return invInfo, nil
}
