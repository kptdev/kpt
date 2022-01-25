// Copyright 2022 Google LLC
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

package internal

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

var functions map[string]framework.ResourceListProcessorFunc = map[string]framework.ResourceListProcessorFunc{
	"gcr.io/kpt-fn/set-labels:v0.1.5": setLabels,
}

func Eval(function string, rl *framework.ResourceList) error {
	if fn, ok := functions[function]; ok {
		return fn(rl)
	} else {
		return fmt.Errorf("unsupported kpt function %q", function)
	}
}
