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
	"strings"
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

func TestVersionCmd_AuthoritativeGitCommit(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	defer func() {
		version = oldVersion
		gitCommit = oldGitCommit
		resetVersionCmdFlags()
	}()
	resetVersionCmdFlags()

	testCommit := "0123456789abcdef0123456789abcdef01234567"
	gitCommit = testCommit
	version = "v1.0.0"

	cmd := GetMain(context.Background())
	cmd.SetArgs([]string{"version"})

	out := captureOutput(func() {
		_ = cmd.Execute()
	})

	assert.Contains(t, out, "Version: v1.0.0")
	assert.Contains(t, out, "Git commit: "+testCommit)
	assert.NotContains(t, out, "(dirty)")
}

func TestVersionCmd_AuthoritativeGitCommitShort(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	defer func() {
		version = oldVersion
		gitCommit = oldGitCommit
		resetVersionCmdFlags()
	}()
	resetVersionCmdFlags()

	testCommit := "0123456789abcdef0123456789abcdef01234567"
	gitCommit = testCommit
	version = "unknown"

	cmd := GetMain(context.Background())
	cmd.SetArgs([]string{"version", "--short"})

	out := captureOutput(func() {
		_ = cmd.Execute()
	})

	assert.Equal(t, "0123456\n", out)
}

func TestVersionCmd_AuthoritativeGitCommitShortWithVersion(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	defer func() {
		version = oldVersion
		gitCommit = oldGitCommit
		resetVersionCmdFlags()
	}()
	resetVersionCmdFlags()

	testCommit := "0123456789abcdef0123456789abcdef01234567"
	gitCommit = testCommit
	version = "v1.0.0"

	cmd := GetMain(context.Background())
	cmd.SetArgs([]string{"version", "--short"})

	out := captureOutput(func() {
		_ = cmd.Execute()
	})

	assert.Equal(t, "v1.0.0\n", out)
}

func TestVersionCmd_EmptyGitCommitFallsBack(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	defer func() {
		version = oldVersion
		gitCommit = oldGitCommit
		resetVersionCmdFlags()
	}()
	resetVersionCmdFlags()

	gitCommit = ""
	version = "v1.0.0"

	cmd := GetMain(context.Background())
	cmd.SetArgs([]string{"version"})

	out := captureOutput(func() {
		_ = cmd.Execute()
	})

	assert.Contains(t, out, "Version: v1.0.0")
	assert.True(t, strings.HasPrefix(out, "Version: v1.0.0\nGit commit: "))
}
