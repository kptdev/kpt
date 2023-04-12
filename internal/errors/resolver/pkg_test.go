// Copyright 2021 The kpt Authors
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

package resolver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/stretchr/testify/assert"
)

func TestPkgErrorResolver(t *testing.T) {
	testCases := map[string]struct {
		err      error
		expected string
	}{
		"kptfileError has nested ErrNotExist": {
			err: &pkg.KptfileError{
				Path: "/foo/bar",
				Err:  os.ErrNotExist,
			},
			expected: "Error: No Kptfile found at \"/foo/bar\".",
		},
		"kptfileError doesn't have a known nested error": {
			err: &pkg.KptfileError{
				Path: "/some/path",
				Err:  fmt.Errorf("this is a test"),
			},
			expected: `
Error: Kptfile at "/some/path" can't be read.

Details:
this is a test
`,
		},
		"kptfileError without nested error": {
			err: &pkg.KptfileError{
				Path: "/some/path",
			},
			expected: "Error: Kptfile at \"/some/path\" can't be read.",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			res, ok := (&pkgErrorResolver{}).Resolve(tc.err)
			if !ok {
				t.Error("expected error to be resolved, but it wasn't")
			}
			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(res.Message))
		})
	}
}
