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

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"sigs.k8s.io/yaml"
)

// Test that the license didn't change for the same module
func TestLicensesForConsistency(t *testing.T) {
	files := make(map[string]*moduleInfo)

	if err := filepath.Walk("modules", func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("error reading file %q: %w", p, err)
		}

		m := &moduleInfo{}
		if err := yaml.Unmarshal(b, m); err != nil {
			return fmt.Errorf("error parsing %q: %w", p, err)
		}

		files[p] = m
		return nil
	}); err != nil {
		t.Fatalf("error during walk: %v", err)
	}

	for f1, m1 := range files {
		dir := filepath.Dir(f1)
		for f2, m2 := range files {
			// Only compare pairs once
			if f1 >= f2 {
				continue
			}

			if filepath.Dir(f2) != dir {
				continue
			}

			if m1.License != m2.License {
				switch f1 {
				case "modules/github.com/klauspost/compress/v1.11.2.yaml":
					// license changed after v1.11.2
				default:
					t.Errorf("license mismatch: %v=%v, %v=%v", f1, m1.License, f2, m2.License)
				}
			}

		}
	}
}
