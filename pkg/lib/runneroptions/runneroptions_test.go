// Copyright 2026 The kpt Authors
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

package runneroptions

import (
	"os"
	"testing"
)

func TestValidatePrefix(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
	}{
		// valid prefixes
		{"empty prefix", "", false},
		{"default GHCR prefix", GHCRImagePrefix, false},
		{"custom registry with org", "my-registry.io/org/", false},
		{"registry with port", "localhost:5000/path/", false},
		{"registry with port no trailing slash", "localhost:5000/path", false},
		{"nested path", "registry.example.com/team/subdir/", false},
		{"simple hostname with path", "gcr.io/my-project/", false},
		{"hyphenated registry", "my-cool-registry.io/functions/", false},
		{"registry with subdomain", "docker.pkg.github.com/owner/repo/", false},
		{"ip address registry", "192.168.1.100:5000/images/", false},
		{"no trailing slash", "ghcr.io/org", false},

		// invalid prefixes
		{"has https scheme", "https://ghcr.io/org/", true},
		{"has http scheme", "http://registry.io/path/", true},
		{"whitespace in prefix", "my registry.io/org/", true},
		{"tab in prefix", "my\tregistry.io/org/", true},
		{"only a slash", "/", true},
		{"only path no host", "/just/a/path/", true},
		{"scheme with bad format", "://bad", true},
		{"fragment in prefix", "registry.io/path#fragment", true},
		{"query string in prefix", "registry.io/path?query=1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := RunnerOptions{ImagePrefix: tt.prefix}
			err := opts.ValidatePrefix()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrefix(%q) error = %v, wantErr %v", tt.prefix, err, tt.wantErr)
			}
		})
	}
}

func TestResolveToImage(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		image    string
		expected string
	}{
		// with default prefix
		{"short image gets prefix", GHCRImagePrefix, "set-namespace:v0.1", "ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.1"},
		{"fully qualified image unchanged", GHCRImagePrefix, "gcr.io/my-project/my-fn:v1", "gcr.io/my-project/my-fn:v1"},
		{"image with org slash unchanged", GHCRImagePrefix, "myorg/my-fn:latest", "myorg/my-fn:latest"},

		// with custom prefix
		{"custom prefix applied", "my-registry.io/functions/", "apply-setters:v0.2", "my-registry.io/functions/apply-setters:v0.2"},
		{"custom prefix no trailing slash", "my-registry.io/functions", "apply-setters:v0.2", "my-registry.io/functions/apply-setters:v0.2"},
		{"custom prefix with fully qualified", "my-registry.io/functions/", "other.io/org/fn:v1", "other.io/org/fn:v1"},

		// with empty prefix
		{"empty prefix returns image as-is", "", "set-namespace:v0.1", "set-namespace:v0.1"},
		{"empty prefix fully qualified unchanged", "", "gcr.io/project/fn:v1", "gcr.io/project/fn:v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := RunnerOptions{ImagePrefix: tt.prefix}
			got := opts.ResolveToImage(tt.image)
			if got != tt.expected {
				t.Errorf("ResolveToImage(%q) with prefix %q = %q, want %q", tt.image, tt.prefix, got, tt.expected)
			}
		})
	}
}

func TestDefaultImagePrefix(t *testing.T) {
	t.Run("returns env var when set", func(t *testing.T) {
		t.Setenv(PrefixEnvVar, "custom-registry.io/functions/")
		got := DefaultImagePrefix()
		if got != "custom-registry.io/functions/" {
			t.Errorf("DefaultImagePrefix() = %q, want %q", got, "custom-registry.io/functions/")
		}
	})

	t.Run("returns GHCR default when env var empty", func(t *testing.T) {
		t.Setenv(PrefixEnvVar, "")
		got := DefaultImagePrefix()
		if got != GHCRImagePrefix {
			t.Errorf("DefaultImagePrefix() = %q, want %q", got, GHCRImagePrefix)
		}
	})

	t.Run("returns GHCR default when env var unset", func(t *testing.T) {
		os.Unsetenv(PrefixEnvVar)
		got := DefaultImagePrefix()
		if got != GHCRImagePrefix {
			t.Errorf("DefaultImagePrefix() = %q, want %q", got, GHCRImagePrefix)
		}
	})
}
