// Copyright 2024-2026 The kpt and Nephio Authors
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

// this code is based on https://github.com/nephio-project/porch/blob/main/pkg/engine/kio.go

package fn

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/kptdev/kpt/api/fn/internal"
	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ParseKubeObjects parses input byte slice to multiple KubeObjects.
func ParseKubeObjects(in []byte) ([]*KubeObject, error) {
	doc, err := internal.ParseDoc(in)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input bytes: %w", err)
	}
	objects, err := doc.Elements()
	if err != nil {
		return nil, fmt.Errorf("failed to extract objects: %w", err)
	}
	var kubeObjects []*KubeObject
	for _, obj := range objects {
		kubeObjects = append(kubeObjects, asKubeObject(obj))
	}
	return kubeObjects, nil
}

// ParseKubeObject parses input byte slice to a single KubeObject.
func ParseKubeObject(in []byte) (*KubeObject, error) {
	objects, err := ParseKubeObjects(in)
	if err != nil {
		return nil, err
	}

	if len(objects) != 1 {
		return nil, fmt.Errorf("expected exactly one object, got %d", len(objects))
	}
	obj := objects[0]
	return obj, nil
}

func ReadKubeObjectsFromDirectory(path string) (KubeObjects, error) {
	reader := &kio.LocalPackageReader{
		PackagePath:           path,
		PackageFileName:       kptfilev1.KptFileName,
		MatchFilesGlob:        MatchAllKRM,
		IncludeSubpackages:    true,
		ErrorIfNonResources:   false,
		OmitReaderAnnotations: false,
		PreserveSeqIndent:     true,
	}
	rnodes, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read KubeObjects from directory %q: %w", path, err)
	}
	kobjs := make([]*KubeObject, len(rnodes))
	for i := range rnodes {
		kobjs[i] = MoveToKubeObject(rnodes[i])
	}
	return kobjs, nil
}

func ReadKubeObjectsFromPackage(inputFiles map[string]string) (objs KubeObjects, extraFiles map[string]string, err error) {
	extraFiles = make(map[string]string)
	for path, content := range inputFiles {
		if !IsKrmResourceFile(path) {
			extraFiles[path] = content
			continue
		}
		fileObjs, err := ReadKubeObjectsFromFile(path, content)
		if err != nil {
			return nil, nil, err
		}
		objs = append(objs, fileObjs...)
	}
	return
}

func ReadKubeObjectsFromFile(filepath string, content string) (KubeObjects, error) {
	reader := &kio.ByteReader{
		Reader: strings.NewReader(content),
		SetAnnotations: map[string]string{
			kioutil.PathAnnotation: filepath,
		},
		DisableUnwrapping: true,
		// need to preserve indentation to avoid Git conflicts in written-out YAML
		PreserveSeqIndent: true,
	}
	nodes, err := reader.Read()
	if err != nil {
		// TODO: fail, or bypass this file too?
		return nil, err
	}
	objs := KubeObjects{}
	for _, node := range nodes {
		objs = append(objs, MoveToKubeObject(node))
	}
	return objs, nil
}

func WriteKubeObjectsToPackage(objs KubeObjects) (map[string]string, error) {
	output := map[string]string{}
	paths := map[string][]*KubeObject{}
	for _, obj := range objs {
		path := PathOfKubeObject(obj)
		paths[path] = append(paths[path], obj)
	}

	var err error
	for path, objs := range paths {
		output[path], err = WriteKubeObjectsToString(objs)
		if err != nil {
			return nil, err
		}
	}
	return output, nil
}

func WriteKubeObjectsToString(objs KubeObjects) (string, error) {
	buf := &bytes.Buffer{}
	bw := kio.ByteWriter{
		Writer: buf,
		ClearAnnotations: []string{
			kioutil.PathAnnotation,
			kioutil.LegacyPathAnnotation, //nolint:staticcheck //SA1019
		},
	}

	nodes := []*yaml.RNode{}
	for _, obj := range objs {
		nodes = append(nodes, obj.CopyToResourceNode())
	}
	if err := bw.Write(nodes); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// PathOfKubeObject returns the path of a KubeObject within a package
// By default is uses the PathAnnotation, otherwise it returns with a default path based on the namespace and name of the object
func PathOfKubeObject(node *KubeObject) string {
	pathAnno := node.PathAnnotation()
	if pathAnno != "" {
		return pathAnno
	}
	ns := node.GetNamespace()
	if ns == "" {
		ns = "no-namespace"
	}
	name := node.GetName()
	if name == "" {
		name = "unnamed"
	}
	return path.Join(ns, fmt.Sprintf("%s.yaml", name))
}

var MatchAllKRM = append([]string{kptfilev1.KptFileName}, kio.MatchAll...)

// IsKrmResourceFile checks if a file in a kpt package should be parsed for KRM resources
func IsKrmResourceFile(path string) bool {
	// Only use the filename for the check for whether we should include the file.
	filename := filepath.Base(path)
	for _, m := range MatchAllKRM {
		if matched, err := filepath.Match(m, filename); err == nil && matched {
			return true
		}
	}
	return false
}
