// Copyright 2026 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package starlark

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
)

func Process(rl *framework.ResourceList) error {
	sr := &Run{}

	if rl.FunctionConfig != nil {
		if err := sr.Config(rl.FunctionConfig); err != nil {
			rl.Results = append(rl.Results, &framework.Result{
				Message:  fmt.Sprintf("failed to configure starlark: %v", err),
				Severity: framework.Error,
			})
			return rl.Results
		}
	}

	if err := sr.Transform(rl); err != nil {
		rl.Results = append(rl.Results, &framework.Result{
			Message:  fmt.Sprintf("starlark transform failed: %v", err),
			Severity: framework.Error,
		})
		return rl.Results
	}

	return nil
}
