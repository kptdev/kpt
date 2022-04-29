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

package engine

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	"github.com/google/go-cmp/cmp"
)

var (
	update = flag.Bool("update", false, "update golden files")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestUnknownBuiltinFunctionMutation(t *testing.T) {
	const doesNotExist = "function-does-not-exist"
	m, err := newBuiltinFunctionMutation(doesNotExist)
	if err == nil || m != nil {
		t.Errorf("creating builtin function runner for an unknown function %q unexpectedly succeeded", doesNotExist)
	}
}

func TestPackageContext(t *testing.T) {
	testdata, err := filepath.Abs(filepath.Join(".", "testdata", "context"))
	if err != nil {
		t.Fatalf("Failed to find testdata: %v", err)
	}

	input := readPackage(t, filepath.Join(testdata, "input"))

	m, err := newBuiltinFunctionMutation(fnruntime.FuncGenPkgContext)
	if err != nil {
		t.Fatalf("Failed to get builtin function mutation: %v", err)
	}

	got, _, err := m.Apply(context.Background(), input)
	if err != nil {
		t.Fatalf("Failed to apply builtin function mutation: %v", err)
	}

	expectedPackage := filepath.Join(testdata, "expected")

	if *update {
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
