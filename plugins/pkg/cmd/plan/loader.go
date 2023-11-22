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

package plan

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

// loadObjectsFromFilesystem parses yaml objects from the provided directory or file path.
func loadObjectsFromFilesystem(p string) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	stat, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("could not get filesystem info for %q: %w", p, err)
	}

	if stat.IsDir() {
		files, err := os.ReadDir(p)
		if err != nil {
			return nil, fmt.Errorf("could not read directory %q: %w", p, err)
		}
		for _, f := range files {
			childP := filepath.Join(p, f.Name())
			childObjects, err := loadObjectsFromFilesystem(childP)
			if err != nil {
				return nil, err
			}
			objects = append(objects, childObjects...)
		}
		return objects, nil
	}

	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("error opening %q: %w", p, err)
	}
	defer f.Close()
	objs, err := parseObjects(f)
	if err != nil {
		return nil, fmt.Errorf("error reading objects: %w", err)
	}
	return objs, nil
}

// parseObjects parses yaml objects from the provided reader.
func parseObjects(r io.Reader) ([]*unstructured.Unstructured, error) {
	codec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	var objs []*unstructured.Unstructured

	scanner := bufio.NewScanner(r)
	scanner.Split(scanYAML)

	for scanner.Scan() {
		b := scanner.Bytes()
		obj, _, err := codec.Decode(b, nil, &unstructured.Unstructured{})
		if err != nil {
			return nil, err
		}
		objs = append(objs, obj.(*unstructured.Unstructured))
	}

	return objs, nil
}

// scanYAML is a split function for a Scanner that returns each yaml object.
func scanYAML(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n---\n")); i >= 0 {
		// We have a full object.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated object. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
