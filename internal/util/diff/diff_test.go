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

// Package diff_test tests the diff package
package diff_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil"
	. "github.com/GoogleContainerTools/kpt/internal/util/diff"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
)

// TestCommand_RunRemoteDiff verifies Command can show changes for remote diff
// operation.
//
// 1. add data to the master branch
// 2. commit and tag the master branch
// 3. add more data to the master branch, commit it
// 4. clone at the tag
// 5. add more data to the master branch, commit it
// 5. Run remote diff between master and cloned
func TestCommand_RunRemoteDiff(t *testing.T) {
	t.SkipNow()
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset3)
	assert.NoError(t, err)
	err = g.Commit("new-data post-v2")
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)
	assert.NotEqual(t, commit, commit2)

	err = get.Command{Git: kptfile.Git{
		Repo: g.RepoDirectory, Ref: "refs/tags/v2", Directory: "/"},
		Destination: filepath.Base(g.RepoDirectory)}.Run()
	assert.NoError(t, err)

	localPkg := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	diffOutput := &bytes.Buffer{}

	err = (&Command{
		Path:         localPkg,
		Ref:          "master",
		DiffType:     "remote",
		DiffTool:     "diff",
		DiffToolOpts: "-r -i -w",
		Output:       diffOutput,
	}).Run()
	assert.NoError(t, err)

	filteredOutput := filterDiffMetadata(diffOutput)

	diffTestOutputDir := filepath.Join(g.DatasetDirectory, testutil.DiffOutput)
	diffOutputGoldenFile := filepath.Join(diffTestOutputDir, "remote_v2_master.txt")

	// If KPT_GENERATE_DIFF_TEST_GOLDEN_FILE env is set, update the golden
	// files.
	if os.Getenv("KPT_GENERATE_DIFF_TEST_GOLDEN_FILE") != "" {
		err = ioutil.WriteFile(diffOutputGoldenFile, []byte(filteredOutput), 0666)
		if err != nil {
			t.Errorf("error writing golden output file: %v", err)
			return
		}
		return
	}
	expOut, err := ioutil.ReadFile(diffOutputGoldenFile)
	assert.NoError(t, err)
	assert.Equal(t, string(expOut), filteredOutput)
}

// TestCommand_RunCombinedDiff verifies Command can show changes for combined diff
// operation.
//
// 1. add data to the master branch
// 2. commit and tag the master branch
// 3. add more data to the master branch, commit it
// 4. clone at the tag
// 5. add more data to the master branch, commit it
// 5. Run combined diff between master and cloned
func TestCommand_RunCombinedDiff(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset3)
	assert.NoError(t, err)
	err = g.Commit("new-data post-v2")
	assert.NoError(t, err)
	commit2, err := g.GetCommit()
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)
	assert.NotEqual(t, commit, commit2)

	err = get.Command{Git: kptfile.Git{
		Repo: g.RepoDirectory, Ref: "refs/tags/v2", Directory: "/"},
		Destination: filepath.Base(g.RepoDirectory)}.Run()
	assert.NoError(t, err)

	localPkg := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	diffOutput := &bytes.Buffer{}

	err = (&Command{
		Path:         localPkg,
		Ref:          "master",
		DiffType:     "combined",
		DiffTool:     "diff",
		DiffToolOpts: "-r -i -w",
		Output:       diffOutput,
	}).Run()
	assert.NoError(t, err)

	filteredOutput := filterDiffMetadata(diffOutput)

	diffTestOutputDir := filepath.Join(g.DatasetDirectory, testutil.DiffOutput)
	diffOutputGoldenFile := filepath.Join(diffTestOutputDir, "combined_v2_master.txt")

	// If KPT_GENERATE_DIFF_TEST_GOLDEN_FILE env is set, update the golden
	// files.
	if os.Getenv("KPT_GENERATE_DIFF_TEST_GOLDEN_FILE") != "" {
		err = ioutil.WriteFile(diffOutputGoldenFile, []byte(filteredOutput), 0666)
		if err != nil {
			t.Errorf("error writing golden output file: %v", err)
			return
		}
		return
	}
	expOut, err := ioutil.ReadFile(diffOutputGoldenFile)
	assert.NoError(t, err)
	assert.Equal(t, string(expOut), filteredOutput)
}

// TestCommand_RunLocalDiff verifies Command can show changes for local diff
// operation.
//
// 1. add data to the master branch
// 2. commit and tag the master branch
// 3. add more data to the master branch, commit it
// 4. clone at the tag
// 5. add more data to the master branch, commit it
// 5. Update cloned package with dataset3
// 6. Run remote diff and verify the output
func TestCommand_Run_LocalDiff(t *testing.T) {
	g, w, clean := testutil.SetupDefaultRepoAndWorkspace(t, testutil.Dataset1)
	defer clean()

	// create a commit with dataset2 and tag it v2, then add another commit on top with dataset3
	commit0, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.ReplaceData(testutil.Dataset2)
	assert.NoError(t, err)
	err = g.Commit("new-data for v2")
	assert.NoError(t, err)
	commit, err := g.GetCommit()
	assert.NoError(t, err)
	err = g.Tag("v2")
	assert.NoError(t, err)
	assert.NotEqual(t, commit, commit0)

	err = get.Command{Git: kptfile.Git{
		Repo: g.RepoDirectory, Ref: "refs/tags/v2", Directory: "/"},
		Destination: filepath.Base(g.RepoDirectory)}.Run()
	assert.NoError(t, err)

	localPkg := filepath.Join(w.WorkspaceDirectory, g.RepoName)

	// make changes in local package
	err = copyutil.CopyDir(filepath.Join(g.DatasetDirectory, testutil.Dataset3), localPkg)
	assert.NoError(t, err)

	diffOutput := &bytes.Buffer{}

	err = (&Command{
		Path:         localPkg,
		Ref:          "master",
		DiffType:     "combined",
		DiffTool:     "diff",
		DiffToolOpts: "-r -i -w",
		Output:       diffOutput,
	}).Run()
	assert.NoError(t, err)

	filteredOutput := filterDiffMetadata(diffOutput)

	diffTestOutputDir := filepath.Join(g.DatasetDirectory, testutil.DiffOutput)
	diffOutputGoldenFile := filepath.Join(diffTestOutputDir, "local_dataset3_v2.txt")

	// If KPT_GENERATE_DIFF_TEST_GOLDEN_FILE env is set, update the golden
	// files.
	if os.Getenv("KPT_GENERATE_DIFF_TEST_GOLDEN_FILE") != "" {
		err = ioutil.WriteFile(diffOutputGoldenFile, []byte(filteredOutput), 0666)
		if err != nil {
			t.Errorf("error writing golden output file: %v", err)
			return
		}
		return
	}
	expOut, err := ioutil.ReadFile(diffOutputGoldenFile)
	assert.NoError(t, err)
	assert.Equal(t, string(expOut), filteredOutput)
}

// filterDiffMetadata removes information from the diff output that is test-run
// specific for ex. removing directory name being used.
func filterDiffMetadata(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	b := &bytes.Buffer{}

	for scanner.Scan() {
		text := scanner.Text()
		// filter out the diff command that contains directory names
		if strings.HasPrefix(text, "diff ") {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n")
	}
	return b.String()
}
