// Copyright 2021,2026 The kpt Authors
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

package render

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/pkg/printer/fake"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCmd_flagAndArgParsing_Symlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()

	if runtime.GOOS == "windows" {
		t.Skip("skipping symlink render test on Windows")
	}

	err := os.MkdirAll(filepath.Join(dir, "path", "to", "pkg", "dir"), 0700)
	assert.NoError(t, err)
	err = os.Symlink(filepath.Join("path", "to", "pkg", "dir"), "foo")
	assert.NoError(t, err)

	// verify the branch ref is set to the correct value
	r := NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
	r.Command.RunE = NoOpRunE
	r.Command.SetArgs([]string{"foo"})
	err = r.Command.Execute()
	assert.NoError(t, err)
	assert.Equal(t, strings.ToLower(filepath.Join("path", "to", "pkg", "dir")), strings.ToLower(r.pkgPath))
}

func TestCmd_timeoutFlag(t *testing.T) {
	testCases := map[string]struct {
		args        []string
		expected    time.Duration
		expectedErr string
	}{
		"not set uses the runtime default": {
			args:     []string{},
			expected: 0,
		},
		"valid duration": {
			args:     []string{"--timeout", "30s"},
			expected: 30 * time.Second,
		},
		"negative duration": {
			args:        []string{"--timeout", "-1s"},
			expectedErr: "invalid timeout -1s: must not be negative",
		},
		"not a duration": {
			args:        []string{"--timeout", "soon"},
			expectedErr: "invalid argument \"soon\" for \"--timeout\" flag",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			r := NewRunner(fake.CtxWithDefaultPrinter(), "kpt")
			r.Command.RunE = NoOpRunE
			r.Command.SilenceErrors = true
			r.Command.SilenceUsage = true
			r.Command.SetArgs(append([]string{t.TempDir()}, tc.args...))

			err := r.Command.Execute()
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, r.RunnerOptions.Timeout)
		})
	}
}

// NoOpRunE is a noop function to replace the run function of a command.  Useful for testing argument parsing.
var NoOpRunE = func(_ *cobra.Command, _ []string) error { return nil }
