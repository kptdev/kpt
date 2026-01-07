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

package builtins

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

type test struct {
	name   string
	dir    string
	expErr error
}

func TestPkgContextGenerator(t *testing.T) {
	tests := []test{
		{
			name: "pkg context should succeed on a non-nested package",
			dir:  "pkg-wo-nesting",
		},
		{
			name: "pkg context should generate on a non-nested package with existing package context",
			dir:  "pkg-with-existing-ctx",
		},
		{
			name: "pkg context should succeed on package with nested package",
			dir:  "pkg-with-nesting",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			pkgCtxGenerator := &PackageContextGenerator{}
			out := &bytes.Buffer{}

			in, err := os.ReadFile(filepath.Join("testdata", test.dir, "in.yaml"))
			assert.NoError(t, err)

			exp, err := os.ReadFile(filepath.Join("testdata", test.dir, "out.yaml"))
			assert.NoError(t, err)

			err = pkgCtxGenerator.Run(bytes.NewReader(in), out)
			if err != test.expErr {
				t.Errorf("exp: %v got: %v", test.expErr, err)
			}
			if diff := cmp.Diff(string(exp), out.String()); diff != "" {
				t.Errorf("pkg context mistmach (-want +got):\n%s", diff)
			}
		})
	}
}
