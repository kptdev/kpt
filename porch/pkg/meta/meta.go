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

import "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"

type PackageMeta struct {
	Tasks       []v1alpha1.Task
	Labels      map[string]string
	Annotations map[string]string
}

func (m *PackageMeta) DeepCopy() *PackageMeta {
	c := &PackageMeta{}
	if m.Tasks != nil {
		c.Tasks = append(c.Tasks, m.Tasks...)
	}
	if m.Labels != nil {
		c.Labels = make(map[string]string)
		for k, v := range m.Labels {
			c.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		c.Annotations = make(map[string]string)
		for k, v := range m.Annotations {
			c.Annotations[k] = v
		}
	}
	return c
}
