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

package packagediscovery

import (
	"reflect"
	"testing"
)

var FilterByPattern = filterByPattern

func TestFilterByPatternForRepositories(t *testing.T) {
	tests := []struct {
		repositories []string
		pattern      string
		want         []string
	}{
		{repositories: []string{"apps", "policies", "namespaces"}, pattern: "*", want: []string{"apps", "policies", "namespaces"}},
		{repositories: []string{"cluster1-apps", "cluster1-policies", "cluster2-apps", "cluster2-policies"}, pattern: "*-apps", want: []string{"cluster1-apps", "cluster2-apps"}},
		{repositories: []string{"cluster1-apps", "cluster1-policies", "cluster2-apps", "cluster2-policies"}, pattern: "cluster1-*", want: []string{"cluster1-apps", "cluster1-policies"}},
		{repositories: []string{"cluster1-apps", "cluster1-policies", "cluster2-apps", "cluster2-policies"}, pattern: "cluster1-apps", want: []string{"cluster1-apps"}},
	}

	for _, tc := range tests {
		got := FilterByPattern(tc.pattern, tc.repositories)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestFilterByPatternForDirectories(t *testing.T) {
	tests := []struct {
		directories []string
		pattern     string
		want        []string
	}{
		{directories: []string{"dev", "prod", "dev/package-a", "dev/package-b", "prod/package-c"}, pattern: "dev/*", want: []string{"dev/package-a", "dev/package-b"}},
		{directories: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "*/dev", want: []string{"package-a/dev", "package-b/dev"}},
		{directories: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "package-*/prod", want: []string{"package-a/prod"}},
		{directories: []string{"package-a", "package-b", "package-a/dev", "package-a/prod", "package-b/dev"}, pattern: "package-a/dev", want: []string{"package-a/dev"}},
		{directories: []string{"parent", "parent/package-a", "parent/package-b", "parent/package-a/dev", "parent/package-a/prod", "parent/package-b/dev"}, pattern: "parent/*-a/dev", want: []string{"parent/package-a/dev"}},
		{directories: []string{"package-a", "package-b", "package-a/crds", "package-a/crs", "package-b/crds"}, pattern: "package-*", want: []string{"package-a", "package-b"}},
	}

	for _, tc := range tests {
		got := FilterByPattern(tc.pattern, tc.directories)
		if !reflect.DeepEqual(tc.want, got) {
			t.Fatalf("expected: %v, got: %v", tc.want, got)
		}
	}
}
