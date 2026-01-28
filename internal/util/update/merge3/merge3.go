// Copyright 2025 The kpt Authors
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

package merge3

import (
	"github.com/kptdev/krm-functions-sdk/go/fn"
	pkgerrors "github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/openapi"
)

func Merge(original, updated, destination fn.KubeObjects, additionalSchemas []byte) (fn.KubeObjects, error) {
	if additionalSchemas != nil {
		if err := openapi.AddSchema(additionalSchemas); err != nil {
			return nil, pkgerrors.Wrap(err, "error adding schema")
		}
	}
	o, u, d := original.CopyToResourceNodes(), updated.CopyToResourceNodes(), destination.CopyToResourceNodes()

	tl := tuples{matcher: &resourceMergeMatcher{}}
	for i := range o {
		if err := tl.addOriginal(o[i]); err != nil {
			return nil, err
		}
	}
	for i := range u {
		if err := tl.addUpdated(u[i]); err != nil {
			return nil, err
		}
	}
	for i := range d {
		if err := tl.addDest(d[i]); err != nil {
			return nil, err
		}
	}
	merged, err := MergeTuples(tl)
	if err != nil {
		return nil, err
	}
	return fn.MoveToKubeObjects(merged), nil
}

func MergeTuples(tl tuples) (kio.ResourceNodeSlice, error) {
	var output kio.ResourceNodeSlice
	for i := range tl.tuplelist {
		t := tl.tuplelist[i]
		strategy := GetHandlingStrategy(t.original, t.updated, t.dest)
		switch strategy {
		case filters.Merge:
			node, err := t.merge()
			if err != nil {
				return nil, err
			}
			if node != nil {
				output = append(output, node)
			}
		case filters.KeepDest:
			output = append(output, t.dest)
		case filters.KeepUpdated:
			output = append(output, t.updated)
		case filters.KeepOriginal:
			output = append(output, t.original)
		case filters.Skip:
			// do nothing
		}
	}
	return output, nil
}
