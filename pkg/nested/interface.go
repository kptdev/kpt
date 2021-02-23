// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package nested

import (
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// NestedInventory represents a package with nested inventory that is read
// from the package manifest.
type NestedInventory struct {
	Path          string
	Resourcegroup *live.InventoryResourceGroup
	Resources     []*unstructured.Unstructured
	Children      []*NestedInventory

	oldChildren []object.ObjMetadata
	newChildren []object.ObjMetadata
}

func (n *NestedInventory) AddChildInventory(u *unstructured.Unstructured, id string) {
	annotations := u.GetAnnotations()
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}
	annotations["config.k8s.io/owning-inventory"] = id
	u.SetAnnotations(annotations)
	meta := object.UnstructuredToObjMeta(u)
	n.newChildren = append(n.newChildren, meta)
}
