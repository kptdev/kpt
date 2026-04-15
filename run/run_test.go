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

package run

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedContain string
	}{
		{
			name:            "semantic version",
			version:         "v1.0.0",
			expectedContain: "kpt version: v1.0.0",
		},
		{
			name:            "development version",
			version:         "v1.0.0-dev",
			expectedContain: "kpt version: v1.0.0-dev",
		},
		{
			name:            "unknown version",
			version:         "unknown",
			expectedContain: "kpt version: unknown (development build)",
		},
		{
			name:            "version with build metadata",
			version:         "v1.0.0+abc123",
			expectedContain: "kpt version: v1.0.0+abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original version
			originalVersion := version
			defer func() { version = originalVersion }()

			// Set test version
			version = tt.version

			// Capture output
			var buf bytes.Buffer
			versionCmd.SetOut(&buf)
			versionCmd.SetErr(&buf)

			// Run command
			err := versionCmd.RunE(versionCmd, []string{})
			if err != nil {
				t.Fatalf("version command failed: %v", err)
			}

			// Check output
			output := buf.String()
			if !strings.Contains(output, tt.expectedContain) {
				t.Errorf("expected output to contain %q, got %q", tt.expectedContain, output)
			}
		})
	}
}

func TestVersionCommandFormat(t *testing.T) {
	// Save original version
	originalVersion := version
	defer func() { version = originalVersion }()

	// Test semantic version format
	version = "v1.0.0"

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	versionCmd.SetErr(&buf)

	err := versionCmd.RunE(versionCmd, []string{})
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()

	// Verify format: "kpt version: vX.Y.Z\n"
	if !strings.HasPrefix(output, "kpt version: v") {
		t.Errorf("expected output to start with 'kpt version: v', got %q", output)
	}

	if !strings.HasSuffix(output, "\n") {
		t.Errorf("expected output to end with newline, got %q", output)
	}
}

func TestVersionCommandUnknown(t *testing.T) {
	// Save original version
	originalVersion := version
	defer func() { version = originalVersion }()

	// Test unknown version
	version = "unknown"

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	versionCmd.SetErr(&buf)

	err := versionCmd.RunE(versionCmd, []string{})
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := buf.String()

	// Verify it shows development build message
	if !strings.Contains(output, "development build") {
		t.Errorf("expected output to contain 'development build' for unknown version, got %q", output)
	}
}
