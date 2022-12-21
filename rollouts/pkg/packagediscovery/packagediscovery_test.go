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

package packagediscovery

import (
	"reflect"
	"testing"
)

var DiscoverPackagePaths = discoverPackagePaths

func TestDiscoverPackagePaths(t *testing.T) {
	tests := []struct {
		paths   []string
		pattern string
		want    []string
	}{
		{paths: []string{"dev", "prod", "dev/package-a", "dev/paackage-b", "prod/package-c"}, pattern: "dev/*", want: []string{"dev/package-a", "dev/package-b"}},
		{paths: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "*/dev", want: []string{"package-a/dev", "package-b/dev"}},
		{paths: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "package-*/prod", want: []string{"package-a/prod"}},
		{paths: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "package-a/dev", want: []string{"package-a/dev"}},
		{paths: []string{"parent", "parent/package-a", "parent/package-b", "parent/package-a/dev", "parent/package-a/prod", "parent/package-b/dev"}, pattern: "parent/*-a/dev", want: []string{"parent/package-a/dev"}},
		{paths: []string{"package-a", "package-b", "package-a/crds", "package-a/crs", "package-b/crds"}, pattern: "package-*", want: []string{"package-a", "package-b"}},
	}

	for _, tc := range tests {
		got := DiscoverPackagePaths(tc.pattern, tc.paths)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}
