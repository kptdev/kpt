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

package git

import "testing"

func TestPackageInDirectory(t *testing.T) {
	for _, tc := range []struct {
		pkg, dir string
		want     bool
	}{
		{
			pkg:  "root/nested",
			dir:  "",
			want: true,
		}, {
			pkg:  "catalog/package",
			dir:  "cat",
			want: false,
		},
		{
			pkg:  "catalog/package",
			dir:  "catalog",
			want: true,
		},
		{
			pkg:  "catalog/package/nested",
			dir:  "catalog/packages",
			want: false,
		},
		{
			pkg:  "catalog/package/nested",
			dir:  "catalog/package",
			want: true,
		},
		{
			pkg:  "catalog/package/nested",
			dir:  "catalog/package/nest",
			want: false,
		},
		{
			pkg:  "catalog/package/nested",
			dir:  "catalog/package/nested",
			want: true,
		},
		{
			pkg:  "catalog/package/nested",
			dir:  "catalog/package/nested/even-more",
			want: false,
		},
	} {
		if got, want := packageInDirectory(tc.pkg, tc.dir), tc.want; got != want {
			t.Errorf("packageInDirectory(%q, %q): got %t, want %t", tc.pkg, tc.dir, got, want)
		}
	}
}
