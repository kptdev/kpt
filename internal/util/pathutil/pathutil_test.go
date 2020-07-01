// Copyright 2020 Google LLC
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

package pathutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"gotest.tools/assert"
)

func TestRel(t *testing.T) {
	base := path.Join(os.TempDir(), "kpt-path-test")
	cwd := base

	actual, err := Rel(base, "resources", cwd)
	expected := "resources"
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = Rel(base, path.Join(base, "resources"), cwd)
	expected = "resources"
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = Rel(base, "./config", cwd)
	expected = "config"
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = Rel(base, "../some-dir", cwd)
	expected = "../some-dir"
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = Rel(base, os.TempDir(), cwd)
	expected = ".."
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = Rel(base, "kpt-path-test", os.TempDir())
	expected = "."
	testutil.AssertNoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestExists(t *testing.T) {
	base, err := ioutil.TempDir("", "kpt-path-test")
	testutil.AssertNoError(t, err)
	defer os.RemoveAll(base)

	assert.Equal(t, Exists(base), true)
	assert.Equal(t, Exists(path.Join(base, "some-random-dir")), false)
}

func TestIsInsideDirectory(t *testing.T) {
	base := path.Join(os.TempDir(), "kpt-path-test")
	file := path.Join(base, "temp-file")
	dirWithSamePrefix := fmt.Sprintf("%s-surfix", base)
	baseWithSeparator := fmt.Sprintf(
		"%s%s",
		base,
		string(os.PathSeparator),
	)

	_, err := IsInsideDir("resources", ".")
	assert.Error(t, err, "argument `path` (resources) is not an absolute path")
	_, err = IsInsideDir("/resources", ".")
	assert.Error(t, err, "argument `directory` (.) is not an absolute path")

	result, err := IsInsideDir(path.Join(base, "."), base)
	assert.NilError(t, err)
	assert.Equal(t, result, true)
	result, err = IsInsideDir(path.Join(base, "."), baseWithSeparator)
	assert.NilError(t, err)
	assert.Equal(t, result, true)
	result, err = IsInsideDir(file, base)
	assert.NilError(t, err)
	assert.Equal(t, result, true)
	result, err = IsInsideDir(path.Join(base, file), baseWithSeparator)
	assert.NilError(t, err)
	assert.Equal(t, result, true)
	result, err = IsInsideDir(path.Join(base, ".."), base)
	assert.NilError(t, err)
	assert.Equal(t, result, false)
	result, err = IsInsideDir(dirWithSamePrefix, base)
	assert.NilError(t, err)
	assert.Equal(t, result, false)
}
