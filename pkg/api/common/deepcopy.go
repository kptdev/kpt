// Copyright 2026 The kpt Authors
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

package common

import (
	"encoding/json"
	"fmt"
	"maps"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func DeepCopyIntoResult(in *framework.Result, out *framework.Result) {
	*out = framework.Result{}
	if in == nil {
		return
	}
	*out = *in
	if in.Tags != nil {
		out.Tags = make(map[string]string, len(in.Tags))
		maps.Copy(out.Tags, in.Tags)
	}
	if in.ResourceRef != nil {
		out.ResourceRef = new(yaml.ResourceIdentifier)
		*out.ResourceRef = *in.ResourceRef
	}
	if in.File != nil {
		out.File = new(framework.File)
		*out.File = *in.File
	}
	if in.Field != nil {
		out.Field = new(framework.Field)
		*out.Field = *in.Field

		if in.Field.CurrentValue != nil {
			out.Field.CurrentValue = DeepCopyInterface(in.Field.CurrentValue)
		}

		if in.Field.ProposedValue != nil {
			out.Field.ProposedValue = DeepCopyInterface(in.Field.ProposedValue)
		}
	}
}

func DeepCopyIntoResults(in *framework.Results, out *framework.Results) {
	if in == nil {
		if out != nil {
			*out = make(framework.Results, 0)
		}
		return
	}
	*out = make(framework.Results, len(*in))
	for i := range *in {
		if (*in)[i] != nil {
			in, out := &(*in)[i], &(*out)[i]
			*out = new(framework.Result)
			// (*in).DeepCopyInto(*out)
			DeepCopyIntoResult(*in, *out)
		}
	}
}

func DeepCopyInterface(in any) any {
	return deepCopyInterface(in, 0)
}

const maxDepth = 1024

func deepCopyInterface(in any, depth uint) any {
	if depth > maxDepth {
		panic(fmt.Sprintf("reached max deepcopy depth of %d", maxDepth))
	}

	// return all primitive / non-pointer types
	switch t := in.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		complex64, complex128,
		bool,
		string,
		json.Number:
		return t
	// inspired by k8s.io/apimachinery/pkg/runtime.DeepCopyJSONValue
	case map[string]any:
		clone := make(map[string]interface{}, len(t))
		for k, v := range t {
			clone[k] = deepCopyInterface(v, depth+1)
		}
		return clone
	case []any:
		clone := make([]interface{}, len(t))
		for i, v := range t {
			clone[i] = deepCopyInterface(v, depth+1)
		}
		return clone
	default:
		panic(fmt.Sprintf("cannot deepcopy type %T", t))
	}
}
