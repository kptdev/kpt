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
	"sort"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// Test that the license didn't change for the same module
func TestLicensesForConsistency(t *testing.T) {
	type moduleVersion struct {
		ModuleInfo *moduleInfo
		Version    string
	}

	modules := make(map[string][]*moduleVersion)

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

		modulePath := filepath.Dir(p)
		modulePath = strings.TrimPrefix(modulePath, "modules/")
		version := filepath.Base(p)
		version = strings.TrimSuffix(version, ".yaml")
		modules[modulePath] = append(modules[modulePath], &moduleVersion{
			ModuleInfo: m,
			Version:    version,
		})
		return nil
	}); err != nil {
		t.Fatalf("error during walk: %v", err)
	}

	for module, versions := range modules {
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].Version < versions[j].Version
		})
		for i := 0; i < len(versions)-1; i++ {
			v1 := versions[i]
			v2 := versions[i+1]

			if v1.ModuleInfo.License != v2.ModuleInfo.License {
				switch module + "@" + v1.Version {
				case "github.com/klauspost/compress@v1.11.2":
					// license changed after v1.11.2
				case "sigs.k8s.io/yaml@v1.3.0":
					// license changed after v1.3.0
				default:
					t.Errorf("license mismatch: %v %v=%v, %v=%v", module, v1.Version, v1.ModuleInfo.License, v2.Version, v2.ModuleInfo.License)
				}
			}

		}
	}
}
