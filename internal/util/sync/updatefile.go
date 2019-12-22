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
	"io/ioutil"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

func SetDependency(dependency kptfile.Dependency) error {
	b, err := ioutil.ReadFile(kptfile.KptFileName)
	if err != nil {
		return errors.WrapPrefixf(err, "failed to read Kptfile -- create one with `kpt init .`")
	}
	k := &kptfile.KptFile{}

	if err := yaml.Unmarshal(b, k); err != nil {
		return errors.WrapPrefixf(err, "failed to unmarshal Kptfile")
	}

	// validate dependencies are well formed
	found := false
	for i := range k.Dependencies {
		d := k.Dependencies[i]
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
		}
		k.Dependencies = append(k.Dependencies, dependency)
	}
	b, err = yaml.Marshal(k)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(kptfile.KptFileName, b, 0600)
}
