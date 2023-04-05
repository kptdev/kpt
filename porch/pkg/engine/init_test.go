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

package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-cmp/cmp"
)

func TestInit(t *testing.T) {
	init := &initPackageMutation{
		name: "testpkg",
		task: &api.Task{
			Init: &api.PackageInitTaskSpec{
				Description: "test package",
				Keywords:    []string{"test", "kpt", "pkg"},
				Site:        "http://kpt.dev/testpkg",
			},
		},
	}

	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "init", "testpkg"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	initializedPkg, _, err := init.Apply(context.Background(), repository.PackageResources{})
	if err != nil {
		t.Errorf("package init failed: %v", err)
	}

	filesToCompare := []string{"Kptfile", "README.md", "package-context.yaml"}

	for _, fi := range filesToCompare {
		got, ok := initializedPkg.Contents[fi]
		if !ok {
			t.Errorf("Cannot find Kptfile in %v", initializedPkg.Contents)
		}

		want, err := os.ReadFile(filepath.Join(testdata, fi))
		if err != nil {
			t.Fatalf("Cannot read expected Kptfile: %v", err)
		}

		if diff := cmp.Diff(string(want), got); diff != "" {
			t.Errorf("Unexpected result (-want, +got): %s", diff)
		}
	}

}
