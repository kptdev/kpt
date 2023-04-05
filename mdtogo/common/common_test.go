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

package common_test

import (
	"os"
	"path"
	"testing"

	"github.com/GoogleContainerTools/kpt/mdtogo/common"
	"github.com/stretchr/testify/assert"
)

func TestReadingMarkdownDirectoryRecursively(t *testing.T) {
	parentTestDir := t.TempDir()
	childTestDir, dirErr := os.MkdirTemp(parentTestDir, "test")
	assert.NoError(t, dirErr)

	firstTestFile, _ := os.Create(path.Join(parentTestDir, "example1.md"))
	secondTestFile, _ := os.Create(path.Join(childTestDir, "example2.md"))
	files, err := common.ReadFiles(parentTestDir, true)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))
	assert.Contains(t, files, firstTestFile.Name())
	assert.Contains(t, files, secondTestFile.Name())
}

func TestReadingMarkdownDirectoryNonrecursively(t *testing.T) {
	testDir := t.TempDir()
	firstTestFile, _ := os.Create(path.Join(testDir, "example1.md"))
	secondTestFile, _ := os.Create(path.Join(testDir, "example2.md"))
	files, err := common.ReadFiles(testDir, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))
	assert.Contains(t, files, firstTestFile.Name())
	assert.Contains(t, files, secondTestFile.Name())
}

func TestReadingMarkdownFileNonrecursively(t *testing.T) {
	testDir := t.TempDir()
	testFile, _ := os.Create(path.Join(testDir, "examples.md"))
	files, err := common.ReadFiles(testFile.Name(), false)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files, testFile.Name())
}

func TestReadingMarkdownFileRecursively(t *testing.T) {
	testDir := t.TempDir()
	testFile, _ := os.Create(path.Join(testDir, "examples.md"))
	files, err := common.ReadFiles(testFile.Name(), true)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Contains(t, files, testFile.Name())
}
