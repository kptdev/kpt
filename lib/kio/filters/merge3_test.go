// Copyright 2019 Google LLC
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

package filters_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"lib.kpt.dev/copyutil"
	"lib.kpt.dev/kio/filters"
)

func TestMerge3_Merge(t *testing.T) {
	_, datadir, _, ok := runtime.Caller(0)
	if !assert.True(t, ok) {
		t.FailNow()
	}
	datadir = filepath.Join(filepath.Dir(datadir), "testdata")

	// setup the local directory
	dir, err := ioutil.TempDir("", "kpt-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer os.RemoveAll(dir)

	if !assert.NoError(t, copyutil.CopyDir(
		filepath.Join(datadir, "dataset1-localupdates"),
		filepath.Join(dir, "dataset1"))) {
		t.FailNow()
	}

	err = filters.Merge3{
		OriginalPath: filepath.Join(datadir, "dataset1"),
		UpdatedPath:  filepath.Join(datadir, "dataset1-remoteupdates"),
		DestPath:     filepath.Join(dir, "dataset1"),
	}.Merge()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	diffs, err := copyutil.Diff(
		filepath.Join(dir, "dataset1"),
		filepath.Join(datadir, "dataset1-expected"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.Empty(t, diffs.List()) {
		t.FailNow()
	}
}
