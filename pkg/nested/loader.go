// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package nested

import (
	"io"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/live"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

// ManifestLoader defines the interface to read the
// package manifests. The manifest can be from either
// the local filesystem or from the stdin.
type ManifestLoader interface {
	Read(io.Reader, []string) (*NestedInventory, error)
}

var _ ManifestLoader = &loader{}

func NewLoader(f util.Factory) *loader {
	return &loader{
		factory: f,
	}
}

type loader struct {
	factory util.Factory
}

func (f *loader) Read(reader io.Reader, args []string) (*NestedInventory, error) {
	l := live.NewResourceGroupManifestLoaderNested(f.factory)
	r, err := l.ManifestReader(reader, args)
	if err != nil {
		return nil, err
	}
	objs, err := r.Read()
	if err != nil {
		return nil, err
	}
	return f.toNestedInventory(objs)
}

func (f *loader) toNestedInventory(objs []*unstructured.Unstructured) (*NestedInventory, error) {
	rgs, nrgs := splitObjects(objs)
	ninv, err := formNestedInventory(rgs)
	if err != nil {
		return nil, err
	}
	err = fillInResources(ninv, nrgs)
	return ninv, err
}

func splitObjects(objs []*unstructured.Unstructured) ([]*unstructured.Unstructured, []*unstructured.Unstructured) {
	var rgs []*unstructured.Unstructured
	var nonRgs []*unstructured.Unstructured
	gvk := schema.GroupVersionKind{
		Group:   kptfile.KptFileGroup,
		Version: kptfile.KptFileVersion,
		Kind:    "ResourceGroup",
	}
	for _, obj := range objs {
		if obj.GroupVersionKind() == gvk {
			rgs = append(rgs, obj)
		} else {
			nonRgs = append(nonRgs, obj)
		}
	}
	return rgs, nonRgs
}

func formNestedInventory(invs []*unstructured.Unstructured) (*NestedInventory, error) {
	if len(invs) == 0 {
		return nil, nil
	}
	visited := make([]bool, len(invs))
	var ninv *NestedInventory

	for i, inv := range invs {
		if inv.GetAnnotations()[kioutil.PathAnnotation] == "" {
			ninv = &NestedInventory{
				Path:          "",
				Resourcegroup: live.WrapInventoryResourceGroup(inv),
				Resources:     nil,
				Children:      nil,
			}
			visited[i] = true
			break
		}
	}
	constructNestedInv(ninv, invs, visited)
	return ninv, nil
}

func constructNestedInv(root *NestedInventory, invs []*unstructured.Unstructured, visited []bool) {
	for i, inv := range invs {
		if visited[i] {
			continue
		}
		path := inv.GetAnnotations()[kioutil.PathAnnotation]
		dir, _ := filepath.Split(path)
		if dir == root.Path {
			node := &NestedInventory{
				Path:          path,
				Resourcegroup: live.WrapInventoryResourceGroup(inv),
				Resources:     nil,
				Children:      nil,
			}
			root.Children = append(root.Children, node)
			root.AddChildInventory(inv, root.Resourcegroup.ID())
			visited[i] = true
			constructNestedInv(node, invs, visited)
		}
	}
	return
}

func fillInResources(ninv *NestedInventory, objs []*unstructured.Unstructured) error {
	if ninv == nil {
		return nil
	}
	for _, obj := range objs {
		if err := fillInSingleResource(ninv, obj); err != nil {
			return err
		}
	}
	return nil
}

func fillInSingleResource(ninv *NestedInventory, obj *unstructured.Unstructured) error {
	path := obj.GetAnnotations()[kioutil.PathAnnotation]
	dir, _ := filepath.Split(path)
	dir = strings.TrimSuffix(dir, string(filepath.Separator))
	if ninv.Path == dir {
		ninv.Resources = append(ninv.Resources, obj)
	} else {
		for _, ch := range ninv.Children {
			if err := fillInSingleResource(ch, obj); err != nil {
				return err
			}
		}
	}
	return nil
}
