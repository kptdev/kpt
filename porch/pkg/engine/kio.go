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

package engine

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type packageReader struct {
	input repository.PackageResources
	extra map[string]string
}

var _ kio.Reader = &packageReader{}

func (r *packageReader) Read() ([]*yaml.RNode, error) {
	results := []*yaml.RNode{}
	for k, v := range r.input.Contents {
		base := path.Base(k)
		ext := path.Ext(base)

		// TODO: use authoritative kpt filtering
		if ext != ".yaml" && ext != ".yml" && base != "Kptfile" {
			r.extra[k] = v
			continue
		}

		var reader kio.Reader = &kio.ByteReader{
			Reader: strings.NewReader(v),
			SetAnnotations: map[string]string{
				kioutil.PathAnnotation: k,
			},
			DisableUnwrapping: true,
		}
		nodes, err := reader.Read()
		if err != nil {
			// TODO: fail, or bypass this file too?
			return nil, err
		}
		results = append(results, nodes...)
	}

	return results, nil
}

type packageWriter struct {
	output repository.PackageResources
}

var _ kio.Writer = &packageWriter{}

func (w *packageWriter) Write(nodes []*yaml.RNode) error {
	paths := map[string][]*yaml.RNode{}
	for _, node := range nodes {
		path := getPath(node)
		paths[path] = append(paths[path], node)
	}

	// TODO: write directly into the package resources abstraction.
	// For now serializing into memory.
	buf := &bytes.Buffer{}
	for path, nodes := range paths {
		bw := kio.ByteWriter{
			Writer: buf,
			ClearAnnotations: []string{
				kioutil.PathAnnotation,
				kioutil.LegacyPathAnnotation,
			},
		}
		if err := bw.Write(nodes); err != nil {
			return err
		}
		w.output.Contents[path] = buf.String()
		buf.Reset()
	}
	return nil
}

func getPath(node *yaml.RNode) string {
	ann := node.GetAnnotations()
	if path, ok := ann[kioutil.PathAnnotation]; ok {
		return path
	}
	ns := node.GetNamespace()
	if ns == "" {
		ns = "non-namespaced"
	}
	name := node.GetName()
	if name == "" {
		name = "unnamed"
	}
	// TODO: harden for escaping etc.
	return path.Join(ns, fmt.Sprintf("%s.yaml", name))
}

type NodeToMapWriter struct {
	Resources map[string]string
}
