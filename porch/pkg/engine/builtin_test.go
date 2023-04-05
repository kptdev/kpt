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

	"github.com/GoogleContainerTools/kpt/internal/builtins"
	"github.com/google/go-cmp/cmp"
)

const (
	updateGoldenFiles = "UPDATE_GOLDEN_FILES"
)

func TestPackageContext(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "context"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	input := readPackage(t, filepath.Join(testdata, "input"))

	packageConfig := &builtins.PackageConfig{
		PackagePath: "parent1/parent1.2/parent1.2.3/me",
	}
	m, err := newPackageContextGeneratorMutation(packageConfig)
	if err != nil {
		t.Fatalf("Failed to get builtin function mutation: %v", err)
	}

	got, _, err := m.Apply(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to apply builtin function mutation: %v", err)
	}

	expectedPackage := filepath.Join(testdata, "expected")

	if os.Getenv(updateGoldenFiles) != "" {
		if err := os.RemoveAll(expectedPackage); err != nil {
			t.Fatalf("Failed to update golden files: %v", err)
		}

		writePackage(t, expectedPackage, got)
	}

	want := readPackage(t, expectedPackage)

	if !cmp.Equal(want, got) {
		t.Errorf("Unexpected result of builtin function mutation (-want, +got): %s", cmp.Diff(want, got))
	}
}
