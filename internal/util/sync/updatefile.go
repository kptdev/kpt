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

package sync

import (
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

func SetDependency(dependency kptfile.Dependency) error {
	k, err := kptfileutil.ReadFile("")
	if err != nil {
		return errors.WrapPrefixf(err, "failed to read Kptfile -- create one with `kpt pkg init .`")
	}

	// validate dependencies are well formed
	found := false
	for i := range k.Dependencies {
		d := &k.Dependencies[i]
		if d.Name != dependency.Name {
			continue
		}
		// update the existing dependency
		if dependency.Strategy != "" {
			d.Strategy = dependency.Strategy
		}
		d.Git.Ref = dependency.Git.Ref
		found = true
		break
	}

	// add the dependency
	if !found {
		if dependency.Strategy == "" {
			dependency.Strategy = string(update.FastForward)
		} else if !update.ValidStrategy(dependency.Strategy) {
			return errors.Errorf("provided update strategy %q is invalid, must "+
				"be one of %q", dependency.Strategy, update.Strategies)
		}
		k.Dependencies = append(k.Dependencies, dependency)
	}

	return kptfileutil.WriteFile("", k)
}
