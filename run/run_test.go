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
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func captureStdout(f func()) string {
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

func TestResolveVersion(t *testing.T) {
	testCases := []struct {
		name          string
		inVersion     string
		inCommit      string
		info          *debug.BuildInfo
		expectedVer   string
		expectedHash  string
		expectedDirty bool
	}{
		{
			name:          "nil build info with unknown version",
			inVersion:     "unknown",
			inCommit:      "",
			info:          nil,
			expectedVer:   "unknown",
			expectedHash:  "unknown",
			expectedDirty: false,
		},
		{
			name:          "empty inVersion and nil info defaults to unknown",
			inVersion:     "",
			inCommit:      "",
			info:          nil,
			expectedVer:   "unknown",
			expectedHash:  "unknown",
			expectedDirty: false,
		},
		{
			name:          "authoritative gitCommit override takes precedence",
			inVersion:     "1.0.0",
			inCommit:      "4600383a5bf70812f9d1bda2670f034cf8daa470",
			info:          nil,
			expectedVer:   "1.0.0",
			expectedHash:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			expectedDirty: false,
		},
		{
			name:      "version set via ldflags and revision in build info",
			inVersion: "1.0.0",
			inCommit:  "",
			info: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "4600383a5bf70812f9d1bda2670f034cf8daa470"},
					{Key: "vcs.modified", Value: "false"},
				},
			},
			expectedVer:   "1.0.0",
			expectedHash:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			expectedDirty: false,
		},
		{
			name:      "version set via ldflags with dirty working tree",
			inVersion: "1.0.0",
			inCommit:  "",
			info: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "4600383a5bf70812f9d1bda2670f034cf8daa470"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			expectedVer:   "1.0.0",
			expectedHash:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			expectedDirty: true,
		},
		{
			name:      "go install with tagged release module version",
			inVersion: "unknown",
			inCommit:  "",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0",
				},
			},
			expectedVer:   "1.0.0",
			expectedHash:  "unknown",
			expectedDirty: false,
		},
		{
			name:      "go install with pseudo-version",
			inVersion: "unknown",
			inCommit:  "",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v0.0.0-20260831115229-4600383a5bf7",
				},
			},
			expectedVer:   "0.0.0-20260831115229-4600383a5bf7",
			expectedHash:  "4600383a5bf7",
			expectedDirty: false,
		},
		{
			name:      "go install with pseudo-version missing v prefix",
			inVersion: "unknown",
			inCommit:  "",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "0.0.0-20260831115229-4600383a5bf7",
				},
			},
			expectedVer:   "0.0.0-20260831115229-4600383a5bf7",
			expectedHash:  "4600383a5bf7",
			expectedDirty: false,
		},
		{
			name:      "release tag with multiple hyphens does not extract fake commit",
			inVersion: "unknown",
			inCommit:  "",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0-rc-abcdefghijkl",
				},
			},
			expectedVer:   "1.0.0-rc-abcdefghijkl",
			expectedHash:  "unknown",
			expectedDirty: false,
		},
		{
			name:      "commit override preserves version fallback from build info",
			inVersion: "unknown",
			inCommit:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0",
				},
			},
			expectedVer:   "1.0.0",
			expectedHash:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			expectedDirty: false,
		},
		{
			name:      "local development build with (devel) module version",
			inVersion: "unknown",
			inCommit:  "",
			info: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "(devel)",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "a0eb64def832fcff168b21d97b1e03da1c3e5956"},
				},
			},
			expectedVer:   "unknown",
			expectedHash:  "a0eb64def832fcff168b21d97b1e03da1c3e5956",
			expectedDirty: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v, hash, dirty := resolveVersion(tc.inVersion, tc.inCommit, tc.info)
			assert.Equal(t, tc.expectedVer, v)
			assert.Equal(t, tc.expectedHash, hash)
			assert.Equal(t, tc.expectedDirty, dirty)
		})
	}
}

func TestVersionCmd_Output(t *testing.T) {
	oldVersion := version
	oldGitCommit := gitCommit
	oldReadBuildInfo := readBuildInfo
	defer func() {
		version = oldVersion
		gitCommit = oldGitCommit
		readBuildInfo = oldReadBuildInfo
		resetVersionCmdFlags()
	}()

	testCases := []struct {
		name        string
		inVersion   string
		inCommit    string
		buildInfo   *debug.BuildInfo
		args        []string
		expectedOut string
	}{
		{
			name:        "authoritative gitCommit override",
			inVersion:   "1.0.0",
			inCommit:    "4600383a5bf70812f9d1bda2670f034cf8daa470",
			buildInfo:   nil,
			args:        []string{"version"},
			expectedOut: "Version: 1.0.0\nGit commit: 4600383a5bf70812f9d1bda2670f034cf8daa470\n",
		},
		{
			name:        "authoritative gitCommit override --short",
			inVersion:   "1.0.0",
			inCommit:    "4600383a5bf70812f9d1bda2670f034cf8daa470",
			buildInfo:   nil,
			args:        []string{"version", "--short"},
			expectedOut: "1.0.0\n",
		},
		{
			name:        "authoritative gitCommit override with unknown version --short",
			inVersion:   "unknown",
			inCommit:    "4600383a5bf70812f9d1bda2670f034cf8daa470",
			buildInfo:   nil,
			args:        []string{"version", "--short"},
			expectedOut: "4600383\n",
		},
		{
			name:      "authoritative gitCommit override with module version in build info",
			inVersion: "unknown",
			inCommit:  "4600383a5bf70812f9d1bda2670f034cf8daa470",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0",
				},
			},
			args:        []string{"version"},
			expectedOut: "Version: 1.0.0\nGit commit: 4600383a5bf70812f9d1bda2670f034cf8daa470\n",
		},
		{
			name:      "go install with tagged release",
			inVersion: "unknown",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0",
				},
			},
			args:        []string{"version"},
			expectedOut: "Version: 1.0.0\nGit commit: unknown\n",
		},
		{
			name:      "go install with tagged release --short",
			inVersion: "unknown",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v1.0.0",
				},
			},
			args:        []string{"version", "--short"},
			expectedOut: "1.0.0\n",
		},
		{
			name:      "go install with pseudo-version",
			inVersion: "unknown",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Main: debug.Module{
					Path:    "github.com/kptdev/kpt",
					Version: "v0.0.0-20260831115229-4600383a5bf7",
				},
			},
			args:        []string{"version"},
			expectedOut: "Version: 0.0.0-20260831115229-4600383a5bf7\nGit commit: 4600383a5bf7\n",
		},
		{
			name:      "official release build with ldflags",
			inVersion: "1.0.0",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "4600383a5bf70812f9d1bda2670f034cf8daa470"},
				},
			},
			args:        []string{"version"},
			expectedOut: "Version: 1.0.0\nGit commit: 4600383a5bf70812f9d1bda2670f034cf8daa470\n",
		},
		{
			name:      "official release build with ldflags --short",
			inVersion: "1.0.0",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "4600383a5bf70812f9d1bda2670f034cf8daa470"},
				},
			},
			args:        []string{"version", "--short"},
			expectedOut: "1.0.0\n",
		},
		{
			name:      "official release build with dirty working tree",
			inVersion: "1.0.0",
			inCommit:  "",
			buildInfo: &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "4600383a5bf70812f9d1bda2670f034cf8daa470"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			args:        []string{"version"},
			expectedOut: "Version: 1.0.0\nGit commit: 4600383a5bf70812f9d1bda2670f034cf8daa470 (dirty)\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetVersionCmdFlags()
			version = tc.inVersion
			gitCommit = tc.inCommit
			readBuildInfo = func() (*debug.BuildInfo, bool) {
				return tc.buildInfo, tc.buildInfo != nil
			}

			cmd := GetMain(context.Background())
			cmd.SetArgs(tc.args)

			out := captureStdout(func() {
				_ = cmd.Execute()
			})

			assert.Equal(t, tc.expectedOut, out)
		})
	}
}
