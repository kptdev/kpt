// Copyright 2023 The kpt Authors
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

package objects

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type Parser struct {
	IncludeLocalConfig bool
}

func (p Parser) AsUnstructureds(resources map[string]string) ([]*unstructured.Unstructured, error) {
	objectList, err := p.AsObjectList(resources)
	if err != nil {
		return nil, err
	}
	unstructureds, err := objectList.AsUnstructured()
	if err != nil {
		return nil, err
	}
	return unstructureds, nil
}

func (p Parser) AsObjectList(resources map[string]string) (*ObjectList, error) {
	var items []*Object

	for filePath, fileContents := range resources {
		ext := path.Ext(filePath)
		ext = strings.ToLower(ext)

		parse := false
		switch ext {
		case ".yaml", ".yml":
			parse = true

		default:
			klog.Warningf("ignoring non-yaml file %s", filePath)
		}

		if !parse {
			continue
		}
		// TODO: Use https://github.com/kubernetes-sigs/kustomize/blob/a5b61016bb40c30dd1b0a78290b28b2330a0383e/kyaml/kio/byteio_reader.go#L170 or similar?
		for _, s := range strings.Split(fileContents, "\n---\n") {
			if isWhitespace(s) {
				continue
			}

			raw := []byte(s)
			u := &unstructured.Unstructured{}
			if err := yaml.Unmarshal(raw, &u); err != nil {
				return nil, fmt.Errorf("error parsing yaml from %s: %w", filePath, err)
			}

			annotations := u.GetAnnotations()

			if !p.IncludeLocalConfig {
				s := annotations["config.kubernetes.io/local-config"]
				if s == "true" {
					continue
				}
			}

			obj := &Object{
				filePath: filePath,
				raw:      raw,
				u:        u,
			}

			// TODO: sync with kpt logic; skip objects marked with the local-only annotation
			items = append(items, obj)
		}
	}

	return &ObjectList{Items: items}, nil
}

func isWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
