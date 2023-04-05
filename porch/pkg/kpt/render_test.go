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

package kpt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/util/render"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func readFile(t *testing.T, path string) []byte {
	if data, err := os.ReadFile(path); err != nil {
		t.Fatalf("Cannot read file %q", err)
		return nil
	} else {
		return data
	}
}

func TestRender(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata"))
	if err != nil {
		t.Fatalf("Cannot compute absolute path for ./testdata: %v", err)
	}

	for _, test := range []struct {
		name string
		pkg  string
		want string
	}{
		{
			name: "render-with-function-config",
			pkg:  "simple-bucket",
			want: "expected.txt",
		},
		{
			name: "render-with-inline-config",
			pkg:  "simple-bucket",
			want: "expected.txt",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			r := render.Renderer{
				PkgPath:    filepath.Join(testdata, test.name, test.pkg),
				Runtime:    &runtime{},
				FileSystem: filesys.FileSystemOrOnDisk{},
				Output:     &output,
			}
			r.RunnerOptions.InitDefaults()

			if _, err := r.Execute(fake.CtxWithDefaultPrinter()); err != nil {
				t.Errorf("Render failed: %v", err)
			}

			got := output.String()
			want := readFile(t, filepath.Join(testdata, test.name, test.want))

			if diff := cmp.Diff(string(want), string(got)); diff != "" {
				t.Errorf("Unexpected result (-want, +got): %s", diff)
			}
		})
	}
}
