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
