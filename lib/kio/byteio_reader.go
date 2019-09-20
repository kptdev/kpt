// Copyright 2019 Google LLC
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

package kio

import (
	"fmt"
	"io"
	"sort"

	"lib.kpt.dev/yaml"
)

// ByteReader decodes ResourceNodes from bytes.
// By default, Read will set the kpt.dev/kio/index annotation on each RNode as it
// is read so they can be written back in the same order.
type ByteReader struct {
	// Reader is where ResourceNodes are decoded from.
	Reader io.Reader

	// OmitReaderAnnotations will configures Read to skip setting the kpt.dev/kio/index
	// annotation on Resources as they are Read.
	OmitReaderAnnotations bool

	// SetAnnotations is a map of caller specified annotations to set on resources as they are read
	// These are independent of the annotations controlled by OmitReaderAnnotations
	SetAnnotations map[string]string
}

var _ Reader = ByteReader{}

func (r ByteReader) Read() ([]*yaml.RNode, error) {
	output := ResourceNodeSlice{}
	decoder := yaml.NewDecoder(r.Reader)
	index := 0
	for {
		node, err := r.decode(index, decoder)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if node == nil {
			// empty value
			continue
		}

		// add the node to the list
		output = append(output, node)

		// increment the index annotation value
		index++
	}
	return output, nil
}

func isEmptyDocument(node *yaml.Node) bool {
	// node is a Document with no content -- e.g. "---\n---"
	return node.Kind == yaml.DocumentNode &&
		node.Content[0].Tag == yaml.NullNodeTag
}

func (r ByteReader) decode(index int, decoder *yaml.Decoder) (*yaml.RNode, error) {
	node := &yaml.Node{}
	err := decoder.Decode(node)
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}

	if isEmptyDocument(node) {
		return nil, nil
	}

	// set annotations on the read Resources
	// sort the annotations by key so the output Resources is consistent (otherwise the
	// annotations will be in a random order)
	n := yaml.NewRNode(node)
	if r.SetAnnotations == nil {
		r.SetAnnotations = map[string]string{}
	}
	if !r.OmitReaderAnnotations {
		r.SetAnnotations[IndexAnnotation] = fmt.Sprintf("%d", index)
	}
	var keys []string
	for k := range r.SetAnnotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, err = n.Pipe(yaml.SetAnnotation(k, r.SetAnnotations[k]))
		if err != nil {
			return nil, err
		}
	}
	return yaml.NewRNode(node), nil
}
