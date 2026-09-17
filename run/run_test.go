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
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func resetVersionCmdFlags() {
	if f := versionCmd.Flags().Lookup("short"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
}

func TestVersionCmd(t *testing.T) {
	const testCommit = "0123456789abcdef0123456789abcdef01234567"

	testCases := []struct {
		name                string
		gitCommit           string
		version             string
		args                []string
		expectedOutput      string
		expectedContains    []string
		expectedNotContains []string
	}{
		{
			name:      "authoritative git commit",
			gitCommit: testCommit,
			version:   "v1.0.0",
			args:      []string{"version"},
			expectedContains: []string{
				"Version: v1.0.0",
				"Git commit: " + testCommit,
			},
			expectedNotContains: []string{
				"(dirty)",
			},
		},
		{
			name:           "authoritative git commit short with unknown version",
			gitCommit:      testCommit,
			version:        "unknown",
			args:           []string{"version", "--short"},
			expectedOutput: "0123456\n",
		},
		{
			name:           "authoritative git commit short with version",
			gitCommit:      testCommit,
			version:        "v1.0.0",
			args:           []string{"version", "--short"},
			expectedOutput: "v1.0.0\n",
		},
		{
			name:      "empty git commit falls back to build info",
			gitCommit: "",
			version:   "v1.0.0",
			args:      []string{"version"},
			expectedContains: []string{
				"Version: v1.0.0\nGit commit: ",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldVersion := version
			oldGitCommit := gitCommit
			defer func() {
				version = oldVersion
				gitCommit = oldGitCommit
				resetVersionCmdFlags()
			}()
			resetVersionCmdFlags()

			gitCommit = tc.gitCommit
			version = tc.version

			cmd := GetMain(context.Background())
			cmd.SetArgs(tc.args)

			out := captureOutput(func() {
				_ = cmd.Execute()
			})

			if tc.expectedOutput != "" {
				assert.Equal(t, tc.expectedOutput, out)
			}
			for _, exp := range tc.expectedContains {
				assert.Contains(t, out, exp)
			}
			for _, notExp := range tc.expectedNotContains {
				assert.NotContains(t, out, notExp)
			}
		})
	}
}
