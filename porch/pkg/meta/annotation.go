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

package meta

import (
	"encoding/json"
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
)

// Annotation is the structured data that we store with commits.
// Currently this is stored as a json-encoded blob in the commit message,
// in future we might use git notes or a similar mechanism.
// TODO: Rationalize with OCI data structure?
type Annotation struct {
	// PackagePath is the path of the package we modified.
	// This is useful for disambiguating which package we are modifying in a tree of packages,
	// without having to check file paths.
	PackagePath string `json:"package,omitempty"`

	// Revision hold the revision of the package revision the commit
	// belongs to.
	Revision string `json:"revision,omitempty"`

	// Task holds the task we performed, if a task caused the commit.
	Task *v1alpha1.Task `json:"task,omitempty"`

	// Annotations holds the annotations.
	// Note: pointer to a map so we can tell the difference between empty and nil
	// TODO: Doesn't workk ... omitempty still includes the null
	Annotations *map[string]string `json:"annotations,omitempty"`

	// Labels holds the labels.
	// Note: pointer to a map so we can tell the difference between empty and nil
	// TODO: Doesn't workk ... omitempty still includes the null
	Labels *map[string]string `json:"labels,omitempty"`
}

func ParseAnnotation(b []byte) (*Annotation, error) {
	annotation := &Annotation{}
	if err := json.Unmarshal(b, annotation); err != nil {
		return nil, fmt.Errorf("error parsing annotation %q: %w", string(b), err)
	}
	return annotation, nil
}
