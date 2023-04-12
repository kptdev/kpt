// Copyright 2019 The kpt Authors
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

package argutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	. "github.com/GoogleContainerTools/kpt/internal/util/argutil"
	"github.com/GoogleContainerTools/kpt/pkg/printer/fake"
	"github.com/stretchr/testify/assert"
)

func TestParseDirVersion(t *testing.T) {
	tests := []struct {
		in     string
		expDir string
		expVer string
		expErr error
	}{
		{ // dir without version
			in:     "dir1",
			expDir: "dir1",
			expVer: "",
			expErr: nil,
		},
		{ // version without dir
			in:     "@ver1",
			expDir: "",
			expVer: "ver1",
			expErr: nil,
		},
		{ // dir with ver
			in:     "/some/dir1@ver1",
			expDir: "/some/dir1",
			expVer: "ver1",
			expErr: nil,
		},
		{ // multiple version
			in:     "/some@dir2@ver1",
			expDir: "",
			expVer: "",
			expErr: ErrMultiVersion,
		},
		{ // empty
			in:     "",
			expDir: "",
			expVer: "",
			expErr: nil,
		},
	}

	for _, test := range tests {
		gotDir, gotVer, gotErr := ParseDirVersion(test.in)

		assert.Equal(t, gotErr, test.expErr)
		assert.Equal(t, gotDir, test.expDir)
		assert.Equal(t, gotVer, test.expVer)
	}
}

func TestParseDirVersionWithDefaults(t *testing.T) {
	tests := []struct {
		in     string
		expDir string
		expVer string
		expErr error
	}{
		{ // dir without version
			in:     "dir1",
			expDir: "dir1",
			expVer: "master",
			expErr: nil,
		},
		{ // version without dir
			in:     "@ver1",
			expDir: "./",
			expVer: "ver1",
			expErr: nil,
		},
		{ // dir with ver
			in:     "/some/dir1@ver1",
			expDir: "/some/dir1",
			expVer: "ver1",
			expErr: nil,
		},
		{ // multiple version
			in:     "/some@dir2@ver1",
			expDir: "",
			expVer: "",
			expErr: ErrMultiVersion,
		},
		{ // empty
			in:     "",
			expDir: "./",
			expVer: "master",
			expErr: nil,
		},
	}

	for _, test := range tests {
		gotDir, gotVer, gotErr := ParseDirVersionWithDefaults(test.in)

		assert.Equal(t, gotErr, test.expErr)
		assert.Equal(t, gotDir, test.expDir)
		assert.Equal(t, gotVer, test.expVer)
	}
}

func TestResolveSymlink(t *testing.T) {
	dir := t.TempDir()
	defer testutil.Chdir(t, dir)()
	err := os.MkdirAll(filepath.Join(dir, "foo"), 0700)
	assert.NoError(t, err)
	err = os.Symlink("foo", "foo-link")
	assert.NoError(t, err)
	err = os.Symlink("foo-link", "link-to-foo-link")
	assert.NoError(t, err)

	actual1, err := ResolveSymlink(fake.CtxWithDefaultPrinter(), "foo-link")
	assert.NoError(t, err)
	assert.Equal(t, "foo", actual1)

	actual2, err := ResolveSymlink(fake.CtxWithDefaultPrinter(), "link-to-foo-link")
	assert.NoError(t, err)
	assert.Equal(t, "foo", actual2)

	actual3, err := ResolveSymlink(fake.CtxWithDefaultPrinter(), ".")
	assert.NoError(t, err)
	assert.Equal(t, ".", actual3)

	_, err = ResolveSymlink(fake.CtxWithDefaultPrinter(), "baz")
	assert.Error(t, err)
	assert.Equal(t, "lstat baz: no such file or directory", err.Error())
}
