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

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

// Simple implementation of set-namespace kpt function, primarily for testing.
func setNamespace(rl *framework.ResourceList) error {
	if rl.FunctionConfig == nil {
		return nil // nothing to do
	}

	if !validGVK(rl.FunctionConfig, "v1", "ConfigMap") {
		return fmt.Errorf("invalid set-namespace function config type: %s/%s; expected v1/ConfigMap", rl.FunctionConfig.GetApiVersion(), rl.FunctionConfig.GetKind())
	}

	data := rl.FunctionConfig.GetDataMap()
	if data == nil {
		return nil // nothing to do
	}

	namespace, ok := data["namespace"]
	if !ok {
		return nil // nothing to do
	}

	for _, n := range rl.Items {
		n.SetNamespace(namespace)
	}

	return nil
}
