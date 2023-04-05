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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
)

func readPackage(t *testing.T, packageDir string) repository.PackageResources {
	results := map[string]string{}

	if err := filepath.Walk(packageDir, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("irregular file object detected: %q (%s)", p, info.Mode())
		}
		rel, err := filepath.Rel(packageDir, p)
		if err != nil {
			return fmt.Errorf("failed to get relative path from %q to %q: %w", packageDir, p, err)
		}
		contents, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("failed to open the source file %q: %w", p, err)
		}
		results[rel] = string(contents)
		return nil
	}); err != nil {
		t.Errorf("Failed to read package from disk %q: %v", packageDir, err)
	}
	return repository.PackageResources{
		Contents: results,
	}
}

func writePackage(t *testing.T, packageDir string, contents repository.PackageResources) {
	for k, v := range contents.Contents {
		abs := filepath.Join(packageDir, k)
		dir := filepath.Dir(abs)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to crete directory %q: %v", dir, err)
		}
		if err := os.WriteFile(abs, []byte(v), 0644); err != nil {
			t.Errorf("Failed to write package file %q: %v", abs, err)
		}
	}
}
