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

package internal

import (
	"fmt"

	function "github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/apply-setters/applysetters"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func applySetters(rl *framework.ResourceList) error {
	if rl.FunctionConfig == nil {
		return nil // nothing to do
	}

	var fn function.ApplySetters
	function.Decode(rl.FunctionConfig, &fn)
	if items, err := fn.Filter(rl.Items); err != nil {
		return fmt.Errorf("apply-setter evaluation failed: %w", err)
	} else {
		rl.Items = items
	}
	return nil
}
