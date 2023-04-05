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
	"errors"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func setLabels(rl *framework.ResourceList) error {
	if rl.FunctionConfig == nil {
		return nil // Done, nothing to do
	}

	var labels map[string]string
	if validGVK(rl.FunctionConfig, "v1", "ConfigMap") {
		labels = rl.FunctionConfig.GetDataMap()
	} else {
		return errors.New("invalid set-labels function config; expected v1/ConfigMap")
	}

	for _, n := range rl.Items {
		l := n.GetLabels()
		for k, v := range labels {
			l[k] = v
		}
		n.SetLabels(l)
	}

	return nil
}

func validGVK(rn *kyaml.RNode, apiVersion, kind string) bool {
	meta, err := rn.GetMeta()
	if err != nil {
		return false
	}
	if meta.APIVersion != apiVersion || meta.Kind != kind {
		return false
	}
	return true
}
