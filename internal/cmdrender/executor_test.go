// Copyright 2021 Google LLC
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

package cmdrender

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestPathRelToRoot(t *testing.T) {
	tests := []struct {
		name         string
		rootPath     string
		subPkgPath   string
		resourcePath string
		expected     string
		errString    string
	}{
		{
			name:         "root package with non absolute path",
			rootPath:     "tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("root package path %q must be absolute", "tmp"),
		},
		{
			name:         "subpackage with non absolute path",
			rootPath:     "/tmp",
			subPkgPath:   "tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage path %q must be absolute", "tmp/a"),
		},
		{
			name:         "resource in a subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "a/c.yaml",
		},
		{
			name:         "resource exists in a deeply nested subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a/b/c",
			resourcePath: "c.yaml",
			expected:     "a/b/c/c.yaml",
		},
		{
			name:         "resource exists in a sub dir with same name as sub package",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "a/c.yaml",
			expected:     "a/a/c.yaml",
		},
		{
			name:         "subpackage is not a descendant of root package",
			rootPath:     "/tmp",
			subPkgPath:   "/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage %q is not a descendant of %q", "/a", "/tmp"),
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			newPath, err := pathRelToRoot(tc.rootPath,
				tc.subPkgPath, tc.resourcePath)
			assert.Equal(t, newPath, tc.expected)
			if tc.errString != "" {
				assert.ErrorContains(t, err, tc.errString)
			}
		})
	}
}
