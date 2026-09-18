// Copyright 2022 The kpt Authors
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

package fn

import (
	"fmt"
	"math"
	"strings"

	kptfileapi "github.com/kptdev/kpt/api/kptfile/v1"
	schema "github.com/kptdev/kpt/api/schema/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KubeObjects []*KubeObject

func (kos KubeObjects) Len() int      { return len(kos) }
func (kos KubeObjects) Swap(i, j int) { kos[i], kos[j] = kos[j], kos[i] }
func (kos KubeObjects) Less(i, j int) bool {
	idi := kos[i].resourceIdentifier()
	idj := kos[j].resourceIdentifier()
	idStrI := fmt.Sprintf("%s %s %s %s", idi.GetAPIVersion(), idi.GetKind(), idi.GetNamespace(), idi.GetName())
	idStrJ := fmt.Sprintf("%s %s %s %s", idj.GetAPIVersion(), idj.GetKind(), idj.GetNamespace(), idj.GetName())
	return idStrI < idStrJ
}

func (kos KubeObjects) String() string {
	var elems []string
	for _, obj := range kos {
		elems = append(elems, strings.TrimSpace(obj.String()))
	}
	return strings.Join(elems, "\n---\n")
}

// EnsureSingleItem checks if KubeObjects contains exactly one item and returns it, or an error if it doesn't.
func (kos KubeObjects) EnsureSingleItem() (*KubeObject, error) {
	if len(kos) == 0 || len(kos) > 1 {
		return nil, fmt.Errorf("%v objects found, but exactly 1 is expected", len(kos))
	}
	return kos[0], nil
}

func (kos KubeObjects) EnsureSingleItemAs(out any) error {
	obj, err := kos.EnsureSingleItem()
	if err != nil {
		return err
	}
	return obj.As(out)
}

// Upsert updates or insert the given KubeObject into the list
// If the list contains an object with the same (Group, Version, Kind, Namespace, Name), then Upsert replaces it with `newObj`,
// otherwise it appends `newObj` to the list
func (kos *KubeObjects) Upsert(newObj *KubeObject) {
	for i, kobj := range *kos {
		if newObj.HasSameID(kobj) {
			(*kos)[i] = newObj
			return
		}
	}
	*kos = append(*kos, newObj)
}

// UpsertTypedObject attempts to convert `newObj` to a KubeObject and then calls Upsert().
func (kos *KubeObjects) UpsertTypedObject(newObj any) error {
	if newObj == nil {
		return fmt.Errorf("obj is nil")
	}

	newKubeObj, err := NewFromTypedObject(newObj)
	if err != nil {
		return err
	}

	kos.Upsert(newKubeObj)
	return nil
}

// Where will return the subset of objects in KubeObjects such that f(object) returns 'true'.
func (kos KubeObjects) Where(f func(*KubeObject) bool) KubeObjects {
	var result KubeObjects
	for _, obj := range kos {
		if f(obj) {
			result = append(result, obj)
		}
	}
	return result
}

// Not returns will return a function that returns the opposite of f(object), i.e. !f(object)
func Not(f func(*KubeObject) bool) func(o *KubeObject) bool {
	return func(o *KubeObject) bool {
		return !f(o)
	}
}

// WhereNot will return the subset of objects in KubeObjects such that f(object) returns 'false'.
// This is a shortcut for Where(Not(f)).
func (kos KubeObjects) WhereNot(f func(o *KubeObject) bool) KubeObjects {
	return kos.Where(Not(f))
}

// Split separates the KubeObjects based on whether the predicate is true or false for them.
func (kos KubeObjects) Split(predicate func(o *KubeObject) bool) (KubeObjects, KubeObjects) {
	tru, fals := KubeObjects{}, KubeObjects{}
	for _, obj := range kos {
		if predicate(obj) {
			tru = append(tru, obj)
		} else {
			fals = append(fals, obj)
		}
	}
	return tru, fals
}

// SetAnnotation sets the specified annotation for all KubeObjects in the slice
func (kos KubeObjects) SetAnnotation(key, value string) error {
	for _, ko := range kos {
		if err := ko.SetAnnotation(key, value); err != nil {
			return err
		}
	}
	return nil
}

// IsGVK returns a function that checks if a KubeObject has a certain GVK.
// Deprecated: Prefer exact matching with IsGroupVersionKind or IsGroupKind
func IsGVK(group, version, kind string) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.IsGVK(group, version, kind)
	}
}

// IsGroupVersionKind returns a function that checks if a KubeObject has a certain GroupVersionKind.
func IsGroupVersionKind(gvk schema.GroupVersionKind) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.IsGroupVersionKind(gvk)
	}
}

// IsGroupKind returns a function that checks if a KubeObject has a certain GroupKind.
func IsGroupKind(gk schema.GroupKind) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.IsGroupKind(gk)
	}
}

// GetRootKptfile returns the root Kptfile. Nested kpt packages can have multiple Kptfile files of the same GVKNN.
func (kos KubeObjects) GetRootKptfile() *KubeObject {
	kptfiles := kos.Where(IsGroupVersionKind(kptfileapi.KptFileGVK()))
	if len(kptfiles) == 0 {
		return nil
	}
	minDepths := math.MaxInt32
	var rootKptfile *KubeObject
	for _, kf := range kptfiles {
		path := kf.PathAnnotation()
		depths := len(strings.Split(path, "/"))
		if depths <= minDepths {
			minDepths = depths
			rootKptfile = kf
		}
	}
	return rootKptfile
}

// IsName returns a function that checks if a KubeObject has a certain name.
func IsName(name string) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.GetName() == name
	}
}

// IsNamespace returns a function that checks if a KubeObject has a certain namespace.
func IsNamespace(namespace string) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.GetNamespace() == namespace
	}
}

// HasLabels returns a function that checks if a KubeObject has all the given labels.
func HasLabels(labels map[string]string) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.HasLabels(labels)
	}
}

// HasAnnotations returns a function that checks if a KubeObject has all the given annotations.
func HasAnnotations(annotations map[string]string) func(*KubeObject) bool {
	return func(o *KubeObject) bool {
		return o.HasAnnotations(annotations)
	}
}

// MoveToKubeObjects moves all yaml.RNodes into KubeObjects, leaving the original slice with empty nodes
func MoveToKubeObjects(rns []*yaml.RNode) KubeObjects {
	var output KubeObjects
	for i := range rns {
		output = append(output, MoveToKubeObject(rns[i]))
	}
	return output
}

// CopyToResourceNodes copies the entire KubeObjects slice to yaml.RNodes
func (kos KubeObjects) CopyToResourceNodes() kio.ResourceNodeSlice {
	var output kio.ResourceNodeSlice
	for i := range kos {
		output = append(output, kos[i].CopyToResourceNode())
	}
	return output
}
