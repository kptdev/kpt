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

package krmvalidate

import (
	"path/filepath"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func writeKptfileForTest(t *testing.T, fsys filesys.FileSystem, dir, name string) {
	t.Helper()
	content := "apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: " + name + "\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "Kptfile"), []byte(content)))
}

func writeValidKRMForTest(t *testing.T, fsys filesys.FileSystem, dir, fileName, resourceName string) {
	t.Helper()
	content := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: " + resourceName + "\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, fileName), []byte(content)))
}

func writeNonKRMForTest(t *testing.T, fsys filesys.FileSystem, dir, fileName string) {
	t.Helper()
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, fileName), []byte("not: krm\njust: [a, plain, map]\nwithout: apiVersion-or-kind\n")))
}

// TestSplit unit-tests the low-level partitioning function directly.
func TestSplit(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	dir := "/pkg"
	assert.NoError(t, fsys.Mkdir(dir))
	writeKptfileForTest(t, fsys, dir, "p")
	writeValidKRMForTest(t, fsys, dir, "a.yaml", "a")
	writeNonKRMForTest(t, fsys, dir, "b.yaml")

	p, err := pkg.New(fsys, dir)
	assert.NoError(t, err)
	nodes, err := p.LocalResources()
	assert.NoError(t, err)

	valid, nonKRM, err := Split(dir, nodes)
	assert.NoError(t, err)
	assert.Len(t, valid, 2) // the Kptfile ("p") plus the ConfigMap ("a")
	var validNames []string
	for _, n := range valid {
		validNames = append(validNames, n.GetName())
	}
	assert.ElementsMatch(t, []string{"p", "a"}, validNames)
	assert.Len(t, nonKRM, 1)
	assert.Equal(t, "b.yaml", nonKRM[0].RelPath)
	assert.Equal(t, dir, nonKRM[0].PkgPath)
}

// TestDecideOutcome_SortedDeterministicMessage verifies the error is
// sorted by pkg path then rel path, with a remediation hint.
func TestDecideOutcome_SortedDeterministicMessage(t *testing.T) {
	found := []NonKRMFile{
		{PkgPath: "/root/z", RelPath: "b.txt", Reason: "boom"},
		{PkgPath: "/root/a", RelPath: "a.txt", Reason: "boom"},
		{PkgPath: "/root/a", RelPath: "z.txt", Reason: "boom"},
	}
	err := DecideOutcome(found, map[string]bool{})
	assert.Error(t, err)
	msg := err.Error()
	idxA1 := indexOf(msg, "a.txt")
	idxA2 := indexOf(msg, "z.txt")
	idxZ := indexOf(msg, "b.txt")
	assert.True(t, idxA1 < idxA2)
	assert.True(t, idxA2 < idxZ)
	assert.Contains(t, msg, AutoIgnoreNonKRMAnnotation)
	assert.Contains(t, msg, ".krmignore")
}

// TestDecideOutcome_OptedInPackageExcused verifies an opted-in
// package's non-KRM file doesn't block, while a non-opted-in one does.
func TestDecideOutcome_OptedInPackageExcused(t *testing.T) {
	found := []NonKRMFile{
		{PkgPath: "/root", RelPath: "blocked.yaml", Reason: "boom"},
		{PkgPath: "/root/sub", RelPath: "excused.yaml", Reason: "boom"},
	}
	optedIn := map[string]bool{"/root/sub": true}
	err := DecideOutcome(found, optedIn)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "blocked.yaml")
	assert.NotContains(t, err.Error(), "excused.yaml")
}

// TestDecideOutcome_NoBlockingFiles_ReturnsNil verifies an all-excused
// or empty found list returns no error.
func TestDecideOutcome_NoBlockingFiles_ReturnsNil(t *testing.T) {
	assert.NoError(t, DecideOutcome(nil, nil))

	found := []NonKRMFile{{PkgPath: "/root", RelPath: "a.yaml", Reason: "boom"}}
	assert.NoError(t, DecideOutcome(found, map[string]bool{"/root": true}))
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
