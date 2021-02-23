// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

// ResourceGroupPathManifestReader encapsulates the default path
// manifest reader.
type ResourceGroupPathManifestReader struct {
	pathReader *manifestreader.PathManifestReader
	nested bool
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
		klog.V(4).Infof(`from Kptfile generating ResourceGroup inventory object "%s/%s/%s"`,
			inv.Namespace, inv.Name, inv.InventoryID)
		if p.nested {
			annotations := invObj.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}
			for k, v := range kf.Annotations {
				if k == kioutil.PathAnnotation {
					v = strings.TrimPrefix(v, p.pathReader.Path)
				}
				annotations[k] = v
			}
			invObj.SetAnnotations(annotations)
		}
		objs = append(objs, invObj)
	} else {
		klog.V(4).Infof("unable to generate ResourceGroup inventory: %s", err)
	}

	// Read Kptfile from the subdirectories
	if p.nested {
		rgs, err := getSubDirResourceGroups(p.pathReader.Path)
		if err != nil {
			klog.V(4).Infof("unable to read the sub package level ResourceGroup: %s", err)
		}
		objs = append(objs, rgs...)
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

func getSubDirResourceGroups(dir string) ([]*unstructured.Unstructured, error) {
	objs := []*unstructured.Unstructured{}

	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if path == dir {
				return nil
			}
			if !info.IsDir() {
				return nil
			}
			kf, err := kptfileutil.ReadFile(path)
			if err != nil {
				klog.V(4).Infof("unable to parse Kptfile for ResourceGroup inventory: %s", err)
				return err
			}
			inv := kf.Inventory
			invObj, err := generateInventoryObj(inv)
			if err == nil {
				klog.V(4).Infof(`from Kptfile generating ResourceGroup inventory object "%s/%s/%s"`,
					inv.Namespace, inv.Name, inv.InventoryID)
				annotations := invObj.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}
				for k, v := range kf.Annotations {
					if k == kioutil.PathAnnotation {
						v = strings.TrimPrefix(strings.TrimPrefix(v, dir), string(filepath.Separator))
					}
					annotations[k] = v
				}
				invObj.SetAnnotations(annotations)
				objs = append(objs, invObj)
			}
			return nil
		})
	return objs, err
}
