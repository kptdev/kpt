// Copyright 2024-2026 The kpt Authors
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

package kptfile

import (
	"fmt"
	"strings"

	kptfileapi "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/api/kubeobject"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	BoolToConditionStatus = map[bool]kptfileapi.ConditionStatus{
		true:  kptfileapi.ConditionTrue,
		false: kptfileapi.ConditionFalse,
	}
)

// KptfileKubeObject provides additional API to manipulate the Kptfile of a kpt package as a KubeObject
type KptfileKubeObject struct {
	kubeobject.KubeObject
}

// NewFromKubeObjectList creates a KptfileKubeObject by finding it in the given KubeObjects list
func NewFromKubeObjectList(objs kubeobject.KubeObjects) (*KptfileKubeObject, error) {
	ko := objs.GetRootKptfile()
	if ko == nil {
		return nil, fmt.Errorf("the Kptfile object is missing from the package")
	}
	return &KptfileKubeObject{KubeObject: *ko}, nil
}

// NewFromPackage creates a KptfileKubeObject from the resource (YAML) files of a package
func NewFromPackage(resources map[string]string) (*KptfileKubeObject, error) {
	kptfileStr, found := resources[kptfileapi.KptFileName]
	if !found {
		return nil, fmt.Errorf("file %q is missing from the package", kptfileapi.KptFileName)
	}

	kos, err := kubeobject.ReadKubeObjectsFromFile(kptfileapi.KptFileName, kptfileStr)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse file %q from package: %w", kptfileapi.KptFileName, err)
	}
	return NewFromKubeObjectList(kos)
}

func NewFromString(str string) (*KptfileKubeObject, error) {
	ko, err := kubeobject.ParseKubeObject([]byte(str))
	if err != nil {
		return nil, err
	}

	if ko.GroupVersionKind() != kptfileapi.KptFileGVK() {
		return nil, fmt.Errorf("string is not Kptfile (GVK %q != %q)", kptfileapi.KptFileGVK(), ko.GroupVersionKind())
	}

	return &KptfileKubeObject{KubeObject: *ko}, nil
}

func (kf *KptfileKubeObject) WriteToPackage(resources map[string]string) error {
	if kf == nil {
		return fmt.Errorf("attempt to write empty Kptfile to the package")
	}
	kptfileStr, err := kubeobject.WriteKubeObjectsToString(kubeobject.KubeObjects{&kf.KubeObject})
	if err != nil {
		return err
	}
	resources[kptfileapi.KptFileName] = kptfileStr
	return nil
}

func (kf *KptfileKubeObject) String() string {
	if kf == nil {
		return ""
	}
	kptfileStr, _ := kubeobject.WriteKubeObjectsToString(kubeobject.KubeObjects{&kf.KubeObject})
	return kptfileStr
}

// Status returns with the Status field of the KptfileKubeObject as a SubObject
// If the Status field doesn't exist, it is added.
func (kf *KptfileKubeObject) Status() *kubeobject.SubObject {
	return kf.UpsertMap("status")
}

// ClearStatus removes the status field from the KptfileKubeObject.
func (kf *KptfileKubeObject) ClearStatus() error {
	if _, err := kf.RemoveNestedField("status"); err != nil {
		return fmt.Errorf("failed to remove status field from Kptfile: %w", err)
	}
	return nil
}

// DecodeKptfile decodes a KptFile from a YAML string.
func DecodeKptfile(kf string) (*kptfileapi.KptFile, error) {
	kptfile := &kptfileapi.KptFile{}
	f := strings.NewReader(kf)
	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err := d.Decode(&kptfile); err != nil {
		return &kptfileapi.KptFile{}, fmt.Errorf("invalid 'v1' Kptfile: %w", err)
	}
	return kptfile, nil
}
