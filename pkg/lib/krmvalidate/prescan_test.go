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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func writeKptfileForTest(t *testing.T, fsys filesys.FileSystem, dir, name string) {
	t.Helper()
	content := "apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: " + name + "\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "Kptfile"), []byte(content)))
}

func writeOptedInKptfileForTest(t *testing.T, fsys filesys.FileSystem, dir, name string) {
	t.Helper()
	content := "apiVersion: kpt.dev/v1\nkind: Kptfile\nmetadata:\n  name: " + name +
		"\n  annotations:\n    " + AutoIgnoreNonKRMAnnotation + ": \"true\"\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "Kptfile"), []byte(content)))
}

// printfCollector returns a printf-shaped func that records every
// formatted line it receives, for asserting on Scan/PrintSkippedSummary output.
func printfCollector() (func(format string, args ...any), *[]string) {
	var lines []string
	return func(format string, args ...any) {
		lines = append(lines, fmt.Sprintf(format, args...))
	}, &lines
}

func writeValidKRMForTest(t *testing.T, fsys filesys.FileSystem, dir string) {
	t.Helper()
	content := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n"
	assert.NoError(t, fsys.WriteFile(filepath.Join(dir, "a.yaml"), []byte(content)))
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
	writeValidKRMForTest(t, fsys, dir)
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

func TestPackageOptedIn(t *testing.T) {
	assert.False(t, PackageOptedIn(nil), "nil Kptfile is never opted in")

	fsys := filesys.MakeFsInMemory()
	assert.NoError(t, fsys.Mkdir("/plain"))
	writeKptfileForTest(t, fsys, "/plain", "plain")
	plainPkg, err := pkg.New(fsys, "/plain")
	assert.NoError(t, err)
	plainKf, err := plainPkg.Kptfile()
	assert.NoError(t, err)
	assert.False(t, PackageOptedIn(plainKf))

	assert.NoError(t, fsys.Mkdir("/optedin"))
	writeOptedInKptfileForTest(t, fsys, "/optedin", "optedin")
	optedInPkg, err := pkg.New(fsys, "/optedin")
	assert.NoError(t, err)
	optedInKf, err := optedInPkg.Kptfile()
	assert.NoError(t, err)
	assert.True(t, PackageOptedIn(optedInKf))
}

func TestScan(t *testing.T) {
	t.Run("clean package: no error, nothing printed, nothing skipped", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeValidKRMForTest(t, fsys, dir)

		printf, printed := printfCollector()
		res, err := Scan(fsys, dir, printf)
		assert.NoError(t, err)
		assert.Empty(t, *printed)
		assert.Empty(t, res.SkippedFiles())
		assert.Contains(t, res.AllResources, dir)
	})

	t.Run("non-KRM file with no annotation blocks with an error", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeNonKRMForTest(t, fsys, dir, "bad.yaml")

		printf, _ := printfCollector()
		res, err := Scan(fsys, dir, printf)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "bad.yaml")
	})

	t.Run("annotated package excuses the non-KRM file and reports the skip", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeOptedInKptfileForTest(t, fsys, dir, "p")
		writeNonKRMForTest(t, fsys, dir, "bad.yaml")

		printf, printed := printfCollector()
		res, err := Scan(fsys, dir, printf)
		assert.NoError(t, err)
		if assert.Len(t, res.SkippedFiles(), 1) {
			assert.Equal(t, "bad.yaml", res.SkippedFiles()[0].RelPath)
		}
		if assert.NotEmpty(t, *printed) {
			assert.Contains(t, (*printed)[0], "bad.yaml")
		}
	})

	t.Run("non-KRM .json file still blocks: render/eval's real reader does read .json", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeNonKRMForTest(t, fsys, dir, "bad.json")

		printf, _ := printfCollector()
		res, err := Scan(fsys, dir, printf)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "bad.json")
	})
}

func TestScanResult_NilReceiver_IsSafe(t *testing.T) {
	var res *ScanResult
	assert.Nil(t, res.SkippedFiles())
	assert.Nil(t, res.SkipFunc("/root"))
}

func TestScanResult_SkipFunc(t *testing.T) {
	res := &ScanResult{skipped: []NonKRMFile{
		{PkgPath: "/root/sub", RelPath: "values.yaml"},
	}}
	skip := res.SkipFunc("/root")
	assert.True(t, skip("sub/values.yaml"))
	assert.False(t, skip("sub/other.yaml"))
	assert.False(t, skip("values.yaml"))

	assert.Nil(t, (&ScanResult{}).SkipFunc("/root"), "no skipped files means no skip func needed")
}

func TestScanTree(t *testing.T) {
	t.Run("single package: non-KRM file found, resources captured", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeValidKRMForTest(t, fsys, dir)
		writeNonKRMForTest(t, fsys, dir, "bad.yaml")

		found, optedIn, allResources, err := ScanTree(fsys, dir, pkg.MatchAllKRM)
		assert.NoError(t, err)
		if assert.Len(t, found, 1) {
			assert.Equal(t, "bad.yaml", found[0].RelPath)
		}
		assert.False(t, optedIn[dir])
		assert.Contains(t, allResources, dir)
	})

	t.Run("subpackages are walked and each tracked by its own opt-in", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/sub"
		assert.NoError(t, fsys.MkdirAll(sub))
		writeKptfileForTest(t, fsys, root, "root")
		writeValidKRMForTest(t, fsys, root)
		writeOptedInKptfileForTest(t, fsys, sub, "sub")
		writeNonKRMForTest(t, fsys, sub, "bad.yaml")

		found, optedIn, allResources, err := ScanTree(fsys, root, pkg.MatchAllKRM)
		assert.NoError(t, err)
		if assert.Len(t, found, 1) {
			assert.Equal(t, sub, found[0].PkgPath)
		}
		assert.False(t, optedIn[root])
		assert.True(t, optedIn[sub])
		assert.Contains(t, allResources, root)
		assert.Contains(t, allResources, sub)
	})

	t.Run("bare directory with no root Kptfile is still scanned", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/bare"
		assert.NoError(t, fsys.Mkdir(dir))
		writeNonKRMForTest(t, fsys, dir, "bad.yaml")

		found, optedIn, _, err := ScanTree(fsys, dir, pkg.MatchAllKRM)
		assert.NoError(t, err)
		assert.Len(t, found, 1)
		assert.False(t, optedIn[dir])
	})

	t.Run("nonexistent target directory propagates an error", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		_, _, _, err := ScanTree(fsys, "/does/not/exist", pkg.MatchAllKRM)
		assert.Error(t, err)
	})

	t.Run("a candidate subpackage with an unparseable Kptfile is silently not discovered, not an error", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/sub"
		assert.NoError(t, fsys.MkdirAll(sub))
		writeKptfileForTest(t, fsys, root, "root")
		assert.NoError(t, fsys.WriteFile(filepath.Join(sub, "Kptfile"), []byte("not: [valid, kptfile\n")))

		found, optedIn, allResources, err := ScanTree(fsys, root, pkg.MatchAllKRM)
		assert.NoError(t, err)
		assert.Empty(t, found)
		assert.Contains(t, allResources, root)
		assert.NotContains(t, allResources, sub, "an unparseable Kptfile means the directory is never recognized as a subpackage")
		assert.NotContains(t, optedIn, sub)
	})
}

func TestScanFlat(t *testing.T) {
	t.Run("root opt-in is reported regardless of where found files actually live", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/sub"
		assert.NoError(t, fsys.MkdirAll(sub))
		writeOptedInKptfileForTest(t, fsys, root, "root")
		writeNonKRMForTest(t, fsys, root, "root-bad.yaml")
		writeKptfileForTest(t, fsys, sub, "sub")
		writeNonKRMForTest(t, fsys, sub, "sub-bad.yaml")

		found, optedIn, allResources, err := ScanFlat(fsys, root)
		assert.NoError(t, err)
		assert.True(t, optedIn)
		assert.Len(t, found, 2)
		assert.NotEmpty(t, allResources)
	})

	t.Run("clean tree returns no found files", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeValidKRMForTest(t, fsys, dir)

		found, optedIn, _, err := ScanFlat(fsys, dir)
		assert.NoError(t, err)
		assert.False(t, optedIn)
		assert.Empty(t, found)
	})

	t.Run("non-KRM .json file is never flagged: the live reader never reads .json", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeValidKRMForTest(t, fsys, dir)
		writeNonKRMForTest(t, fsys, dir, "unrelated.json")

		found, optedIn, _, err := ScanFlat(fsys, dir)
		assert.NoError(t, err)
		assert.False(t, optedIn)
		assert.Empty(t, found)
	})
}

func TestScanFlatAndDecide(t *testing.T) {
	t.Run("blocking non-KRM file returns an error, nothing printed", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeNonKRMForTest(t, fsys, dir, "bad.yaml")

		printf, printed := printfCollector()
		res, err := ScanFlatAndDecide(fsys, dir, printf)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "bad.yaml")
		assert.Empty(t, *printed)
	})

	t.Run("excused subpackage file is reported and skipped", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		root := "/root"
		sub := "/root/sub"
		assert.NoError(t, fsys.MkdirAll(sub))
		writeKptfileForTest(t, fsys, root, "root")
		writeOptedInKptfileForTest(t, fsys, sub, "sub")
		writeNonKRMForTest(t, fsys, sub, "bad.yaml")

		printf, printed := printfCollector()
		res, err := ScanFlatAndDecide(fsys, root, printf)
		assert.NoError(t, err)
		if assert.Len(t, res.SkippedFiles(), 1) {
			assert.Equal(t, "bad.yaml", res.SkippedFiles()[0].RelPath)
		}
		assert.NotEmpty(t, *printed)
	})

	t.Run("clean tree returns an empty result, nothing printed", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeValidKRMForTest(t, fsys, dir)

		printf, printed := printfCollector()
		res, err := ScanFlatAndDecide(fsys, dir, printf)
		assert.NoError(t, err)
		assert.Empty(t, res.SkippedFiles())
		assert.Empty(t, *printed)
	})

	t.Run("non-KRM .json file does not block: the live reader never reads .json", func(t *testing.T) {
		fsys := filesys.MakeFsInMemory()
		dir := "/pkg"
		assert.NoError(t, fsys.Mkdir(dir))
		writeKptfileForTest(t, fsys, dir, "p")
		writeNonKRMForTest(t, fsys, dir, "unrelated.json")

		printf, printed := printfCollector()
		res, err := ScanFlatAndDecide(fsys, dir, printf)
		assert.NoError(t, err)
		assert.Empty(t, res.SkippedFiles())
		assert.Empty(t, *printed)
	})
}

func TestFlatScanResult_NilReceiver_IsSafe(t *testing.T) {
	var res *FlatScanResult
	assert.Nil(t, res.SkippedFiles())
	assert.Nil(t, res.SkipFunc("/root"))
}

func TestFlatScanResult_SkipFunc_ReRootsSubpackagePaths(t *testing.T) {
	res := &FlatScanResult{skipped: []NonKRMFile{
		{PkgPath: "/root/sub", RelPath: "values.yaml"},
	}}
	skip := res.SkipFunc("/root")
	assert.True(t, skip("sub/values.yaml"), "a subpackage file's skip path must be re-rooted against pkgRootPath")
	assert.False(t, skip("values.yaml"))

	assert.Nil(t, (&FlatScanResult{}).SkipFunc("/root"))
}

func TestCachedLocalResources(t *testing.T) {
	fsys := filesys.MakeFsInMemory()
	dir := "/pkg"
	assert.NoError(t, fsys.Mkdir(dir))
	writeKptfileForTest(t, fsys, dir, "p")
	writeValidKRMForTest(t, fsys, dir)

	p, err := pkg.New(fsys, dir)
	assert.NoError(t, err)

	t.Run("cache miss falls back to a real read", func(t *testing.T) {
		resources, err := CachedLocalResources(p, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, resources)
	})

	t.Run("cache hit returns the cached slice, not a fresh read", func(t *testing.T) {
		cache := map[string][]*yaml.RNode{p.UniquePath.String(): {}} // deliberately empty, unlike a real read
		resources, err := CachedLocalResources(p, cache)
		assert.NoError(t, err)
		assert.Empty(t, resources, "must return the cached (empty) slice, not re-read from disk")
	})
}

func TestPrintSkippedSummary(t *testing.T) {
	t.Run("nothing skipped: no-op", func(t *testing.T) {
		printf, printed := printfCollector()
		PrintSkippedSummary(printf, nil, nil)
		assert.Empty(t, *printed)
	})

	t.Run("multiple opted-in packages are printed once each, sorted by path", func(t *testing.T) {
		found := []NonKRMFile{
			{PkgPath: "/root/z", RelPath: "b.yaml"},
			{PkgPath: "/root/a", RelPath: "a.yaml"},
			{PkgPath: "/root/a", RelPath: "c.yaml"},
			{PkgPath: "/root/not-opted-in", RelPath: "skip-me.yaml"},
		}
		optedIn := map[string]bool{"/root/z": true, "/root/a": true}

		printf, printed := printfCollector()
		PrintSkippedSummary(printf, found, optedIn)

		if assert.Len(t, *printed, 2) {
			assert.Contains(t, (*printed)[0], "/root/a")
			assert.Contains(t, (*printed)[0], "a.yaml")
			assert.Contains(t, (*printed)[0], "c.yaml")
			assert.Contains(t, (*printed)[1], "/root/z")
			assert.Contains(t, (*printed)[1], "b.yaml")
		}
		for _, line := range *printed {
			assert.NotContains(t, line, "skip-me.yaml")
		}
	})
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
