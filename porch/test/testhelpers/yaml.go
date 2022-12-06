package testhelpers

/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"bytes"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlserializer "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

func (h *Harness) ParseObjects(y string) []*unstructured.Unstructured {
	t := h.T

	var objects []*unstructured.Unstructured

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(y)), 100)
	for {
		var rawObj runtime.RawExtension
		if err := decoder.Decode(&rawObj); err != nil {
			if err != io.EOF {
				t.Fatalf("error decoding yaml: %v", err)
			}
			break
		}

		m, _, err := yamlserializer.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			t.Fatalf("error decoding yaml: %v", err)
		}

		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(m)
		if err != nil {
			t.Fatalf("error parsing object: %v", err)
		}
		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		objects = append(objects, unstructuredObj)
	}

	return objects
}

func (h *Harness) ParseOneObject(y string) *unstructured.Unstructured {
	objects := h.ParseObjects(y)
	if len(objects) != 1 {
		h.Fatalf("expected exactly one object, got %d", len(objects))
	}
	return objects[0]
}

func ToYAML[T any](h *Harness, items []T) string {
	var s []string
	for i := range items {
		item := &items[i]
		b, err := yaml.Marshal(item)
		if err != nil {
			h.Fatalf("failed to marshal object to yaml: %v", err)
			return ""
		}
		s = append(s, string(b))
	}
	return strings.Join(s, "\n---\n")
}
