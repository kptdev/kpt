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

package runner

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSanitizeTimestamps(t *testing.T) {
	grid := []struct {
		Name   string
		Input  string
		Output string
	}{
		{
			Name: "Prefix match: 12s and 12.1s",
			Input: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 12s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" in 12.1s
`,
			Output: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 0s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" in 0s
`,
		},
		{
			Name: "Suffix match: 1s and 0.1s",
			Input: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 1s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" in 0.1s
`,
			Output: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 0s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" in 0s
`,
		},
		{
			Name: "Only substitute matching lines",
			Input: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 1s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:1s\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" notin 1s
`,
			Output: `
[RUNNING] \"gcr.io/kpt-fn/starlark:v0.2.1\"
[PASS] \"gcr.io/kpt-fn/starlark:v0.2.1\" in 0s
[RUNNING] \"gcr.io/kpt-fn/set-namespace:1s\" on 1 resource(s)
[PASS] \"gcr.io/kpt-fn/set-namespace:v0.1.3\" notin 1s
`,
		},
	}

	for _, g := range grid {
		g := g // Avoid range go-tcha
		t.Run(g.Name, func(t *testing.T) {
			got := sanitizeTimestamps(g.Input)
			want := g.Output

			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("unexpected results (-want, +got): %s", diff)
			}
		})
	}
}
