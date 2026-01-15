// Copyright 2023 The kpt Authors
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

package pkgbuilder

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestBuildKptfile(t *testing.T) {
	var repos ReposInfo
	pkg := &pkg{}
	pkg.Kptfile = &Kptfile{
		Pipeline: &Pipeline{
			Functions: []Function{
				{Image: "example.com/fn1"},
				{ConfigPath: "config1"},
				{Image: "example.com/fn2"},
			},
		},
	}

	got := buildKptfile(pkg, "test1", repos)
	want := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test1
pipeline:
  mutators:
  - image: example.com/fn1
  - configPath: config1
  - image: example.com/fn2
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("buildKptfile returned unexpected diff (-want +got):\n%s", diff)
	}
}
