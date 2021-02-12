// Copyright 2020 Google LLC
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

// Package pipeline provides struct definitions for Pipeline and utility
// methods to read and write a pipeline resource.
package pipeline

import (
	"github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

const (
	sourceAllSubPkgs string = "./*"
	sourceCurrentPkg string = "."
)

// fnChain returns a slice of function runners from the
// functions and configs defined in pipeline.
func fnChain(p *v1alpha2.Pipeline, pkgPath string) ([]kio.Filter, error) {
	fns := []v1alpha2.Function{}
	fns = append(fns, p.Mutators...)
	// TODO: Validators cannot modify resources.
	fns = append(fns, p.Validators...)
	var runners []kio.Filter
	for i := range fns {
		fn := fns[i]
		r, err := newFnRunner(&fn, pkgPath)
		if err != nil {
			return nil, err
		}
		runners = append(runners, r)
	}
	return runners, nil
}
