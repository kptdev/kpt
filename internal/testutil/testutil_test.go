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

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSyncWritableFile copies a read-only source (as served from the module
// cache) and checks the destination is owner-writable, the executable bit is
// preserved, and the source mode is untouched.
func TestSyncWritableFile(t *testing.T) {
	root := t.TempDir()

	cases := map[string]os.FileMode{
		"plain.yaml": 0444, // read-only, no exec
		"script.sh":  0555, // read-only, exec
	}
	for name, srcMode := range cases {
		t.Run(name, func(t *testing.T) {
			src := filepath.Join(root, "src-"+name)
			require.NoError(t, os.WriteFile(src, []byte("data\n"), srcMode))

			dst := filepath.Join(root, "dst-"+name)
			require.NoError(t, syncWritableFile(src, dst))

			dstMode := mode(t, dst)
			assert.NotZero(t, dstMode&0200, "destination should be owner-writable")
			assert.Equal(t, srcMode&0100, dstMode&0100, "executable bit should be preserved")
			assert.Equal(t, srcMode, mode(t, src), "source mode should be unchanged")
		})
	}
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}
