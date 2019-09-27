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

package walk

import "lib.kpt.dev/yaml"

// walkScalar calls SetScalarValue and sets the value on the dest
func (l Filter) walkScalar(dest *yaml.RNode) error {
	if err := l.SetComments(l.Source, dest); err != nil {
		return err
	}

	r, err := l.SetScalarValue(l.Source, dest)
	if err != nil || r == nil {
		return err
	}
	dest.SetYNode(r.YNode())
	return err
}
