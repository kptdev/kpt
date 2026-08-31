// Copyright 2021,2026 The kpt Authors
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

package pkg_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/internal/testutil"
	"github.com/kptdev/kpt/internal/testutil/pkgbuilder"
	. "github.com/kptdev/kpt/pkg/lib/pkg"
	"github.com/stretchr/testify/assert"
)

func TestWalkPackage(t *testing.T) {
	testCases := map[string]struct {
		pkg      *pkgbuilder.RootPkg
		expected []string
	}{
		"walks subdirectories of a package": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithFile("def.yaml", "123"),
				),
			expected: []string{
				".",
				"abc.yaml",
				"foo",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"ignores .git folder": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("abc.yaml", "42").
				WithSubPackages(
					pkgbuilder.NewSubPkg(".git").
						WithFile("INDEX", "ABC123"),
				),
			expected: []string{
				".",
				"abc.yaml",
			},
		},
		"ignores subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
				),
			expected: []string{
				".",
				"abc.yaml",
				"test.txt",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, testutil.EmptyReposInfo)

			var visited []string
			if err := WalkPackage(pkgPath, func(s string, _ os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(pkgPath, s)
				if err != nil {
					return err
				}
				visited = append(visited, relPath)
				return nil
			}); !assert.NoError(t, err) {
				t.FailNow()
			}

			sort.Strings(visited)

			assert.Equal(t, tc.expected, visited)
		})
	}
}

func TestCopyPackage(t *testing.T) {
	testCases := map[string]struct {
		pkg               *pkgbuilder.RootPkg
		copyRootKptfile   bool
		subpackageMatcher SubpackageMatcher
		expected          []string
	}{
		"subpackages without root kptfile": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
				),
			copyRootKptfile:   false,
			subpackageMatcher: Local,
			expected: []string{
				".",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"ignores .git folder": {
			pkg: pkgbuilder.NewRootPkg().
				WithFile("abc.yaml", "42").
				WithSubPackages(
					pkgbuilder.NewSubPkg(".git").
						WithFile("INDEX", "ABC123"),
				),
			subpackageMatcher: None,
			expected: []string{
				".",
				"abc.yaml",
			},
		},
		"ignore subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
				),
			copyRootKptfile:   true,
			subpackageMatcher: None,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"test.txt",
			},
		},
		"include all subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
						WithFile("def.yaml", "123"),
				),
			copyRootKptfile:   true,
			subpackageMatcher: All,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"bar",
				"bar/Kptfile",
				"bar/def.yaml",
				"foo",
				"foo/Kptfile",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"include only local subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
						WithFile("def.yaml", "123"),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Local,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"include only remote subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123"),
					pkgbuilder.NewSubPkg("bar").
						WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
						WithFile("def.yaml", "123"),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Remote,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"bar",
				"bar/Kptfile",
				"bar/def.yaml",
				"test.txt",
			},
		},
		"include local subpackage with remote child": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123").WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
							WithFile("def.yaml", "123"),
					),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Local,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/bar",
				"foo/bar/Kptfile",
				"foo/bar/def.yaml",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"include local subpackage with local child": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithFile("def.yaml", "123").WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile().
							WithFile("def.yaml", "123"),
					),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Local,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/bar",
				"foo/bar/Kptfile",
				"foo/bar/def.yaml",
				"foo/def.yaml",
				"test.txt",
			},
		},
		"include remote subpackage with local child": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
						WithFile("def.yaml", "123").WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile().
							WithFile("def.yaml", "123"),
					),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Remote,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/def.yaml",
				"foo/bar",
				"foo/bar/Kptfile",
				"foo/bar/def.yaml",
				"test.txt",
			},
		},
		"include remote subpackage with remote child": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithFile("abc.yaml", "42").
				WithFile("test.txt", "Hello, World!").
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
						WithFile("def.yaml", "123").WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(pkgbuilder.NewKptfile().WithUpstream("", "", "", "")).
							WithFile("def.yaml", "123"),
					),
				),
			copyRootKptfile:   true,
			subpackageMatcher: Remote,
			expected: []string{
				".",
				"Kptfile",
				"abc.yaml",
				"foo",
				"foo/Kptfile",
				"foo/def.yaml",
				"foo/bar",
				"foo/bar/Kptfile",
				"foo/bar/def.yaml",
				"test.txt",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, testutil.EmptyReposInfo)
			dest := t.TempDir()

			err := CopyPackage(pkgPath, dest, tc.copyRootKptfile, tc.subpackageMatcher)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var visited []string
			if err = filepath.Walk(dest, func(s string, _ os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				relPath, err := filepath.Rel(dest, s)
				if err != nil {
					return err
				}
				visited = append(visited, relPath)
				return nil
			}); !assert.NoError(t, err) {
				t.FailNow()
			}

			sort.Strings(visited)

			assert.ElementsMatch(t, tc.expected, visited)
		})
	}
}

func TestFindLocalRecursiveSubpackagesForPaths(t *testing.T) {
	testCases := map[string]struct {
		pkgs     []*pkgbuilder.RootPkg
		expected []string
	}{
		"does not include remote subpackages": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstream("github.com/kptdev/kpt",
										"/", "main", string(kptfilev1.ResourceMerge)),
							).
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expected: []string{},
		},
		"includes local subpackages": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource),
					),
			},
			expected: []string{
				"foo",
			},
		},
		"includes root package": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithKptfile().
					WithResource(pkgbuilder.DeploymentResource),
			},
			expected: []string{},
		},
		"does not include nested remote subpackages": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstream("github.com/kptdev/kpt",
										"/", "main", string(kptfilev1.ResourceMerge)),
							).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithSubPackages(
										pkgbuilder.NewSubPkg("zork").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstream("github.com/kptdev/kpt",
														"/", "main", string(kptfilev1.ResourceMerge)),
											).
											WithResource(pkgbuilder.ConfigMapResource),
									),
							),
					),
			},
			expected: []string{},
		},
		"includes nested local subpackages": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("zork").
									WithKptfile().
									WithResource(pkgbuilder.ConfigMapResource),
							),
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(),
					),
			},
			expected: []string{
				"foo",
				"foo/zork",
				"subpkg",
			},
		},
		"multiple packages": {
			pkgs: []*pkgbuilder.RootPkg{
				pkgbuilder.NewRootPkg().
					WithResource(pkgbuilder.DeploymentResource).
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile().
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("zork").
									WithKptfile().
									WithResource(pkgbuilder.ConfigMapResource),
							),
						pkgbuilder.NewSubPkg("subpkg").
							WithKptfile(),
					),
				pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("foo").
							WithKptfile(),
					),
				pkgbuilder.NewRootPkg().
					WithKptfile().
					WithSubPackages(
						pkgbuilder.NewSubPkg("bar").
							WithKptfile(),
						pkgbuilder.NewSubPkg("remotebar").
							WithKptfile(
								pkgbuilder.NewKptfile().
									WithUpstream("github.com/kptdev/kpt",
										"/", "main", string(kptfilev1.ResourceMerge)),
							),
					),
			},
			expected: []string{
				"bar",
				"foo",
				"foo/zork",
				"subpkg",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pkgPaths []string
			for _, p := range tc.pkgs {
				pkgPaths = append(pkgPaths, p.ExpandPkg(t, testutil.EmptyReposInfo))
			}

			paths, err := FindSubpackagesForPaths(Local, true, pkgPaths...)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			sort.Strings(paths)

			assert.Equal(t, tc.expected, paths)
		})
	}
}

func TestRemoveStaleItems_RemovesFile(t *testing.T) {
	org := t.TempDir()
	src := t.TempDir()
	dst := t.TempDir()

	// Create a file in org and dst, but not in src
	fileName := "file.txt"
	assert.NoError(t, os.WriteFile(filepath.Join(org, fileName), []byte("content"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(dst, fileName), []byte("content"), 0644))

	// Should remove file.txt from dst
	err := RemoveStaleItems(org, src, dst, true, All)
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dst, fileName))
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveStaleItems_PreservesNonEmptyLocalDir(t *testing.T) {
	org := t.TempDir()
	src := t.TempDir()
	dst := t.TempDir()

	fileName := "file.txt"
	filePathDst := filepath.Join(dst, fileName)
	filePathOrg := filepath.Join(org, fileName)

	assert.NoError(t, os.WriteFile(filePathOrg, []byte("content"), 0644))
	assert.NoError(t, os.WriteFile(filePathDst, []byte("content"), 0644))

	// Replace file in dst with a non-empty directory (simulates upstream deleting
	// a path that locally became a directory with added files).
	assert.NoError(t, os.Remove(filePathDst))
	assert.NoError(t, os.Mkdir(filePathDst, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(filePathDst, "dummy"), []byte("x"), 0644))

	// RemoveStaleItems should succeed and preserve the non-empty directory.
	err := RemoveStaleItems(org, src, dst, true, All)
	assert.NoError(t, err)

	// The directory and its locally-added file must still exist.
	_, err = os.Stat(filePathDst)
	assert.NoError(t, err, "non-empty stale directory should be preserved")
	_, err = os.Stat(filepath.Join(filePathDst, "dummy"))
	assert.NoError(t, err, "locally-added file inside stale directory should be preserved")
}

func TestRemoveStaleItems_PreservesLocalFilesInDeletedDir(t *testing.T) {
	org := t.TempDir()
	src := t.TempDir()
	dst := t.TempDir()

	// Simulate: origin has configs/base.yaml, upstream deletes entire configs/ dir,
	// but local added configs/custom.yaml.
	configsDirOrg := filepath.Join(org, "configs")
	configsDirDst := filepath.Join(dst, "configs")

	assert.NoError(t, os.Mkdir(configsDirOrg, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(configsDirOrg, "base.yaml"), []byte("original"), 0644))

	assert.NoError(t, os.Mkdir(configsDirDst, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(configsDirDst, "base.yaml"), []byte("original"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(configsDirDst, "custom.yaml"), []byte("local-addition"), 0644))

	// src (new upstream) has no configs/ directory at all.

	err := RemoveStaleItems(org, src, dst, true, All)
	assert.NoError(t, err)

	// base.yaml was in origin and not in upstream — should be removed.
	_, err = os.Stat(filepath.Join(configsDirDst, "base.yaml"))
	assert.True(t, os.IsNotExist(err), "stale file base.yaml should be removed")

	// custom.yaml was NOT in origin — should be preserved.
	_, err = os.Stat(filepath.Join(configsDirDst, "custom.yaml"))
	assert.NoError(t, err, "locally-added file custom.yaml should be preserved")

	// configs/ directory should still exist because it contains custom.yaml.
	info, err := os.Stat(configsDirDst)
	assert.NoError(t, err, "directory with local files should be preserved")
	assert.True(t, info.IsDir())
}

func TestRemoveStaleItems_RemovesEmptyDirAfterStaleFileCleanup(t *testing.T) {
	org := t.TempDir()
	src := t.TempDir()
	dst := t.TempDir()

	// Simulate: origin has configs/base.yaml, upstream deletes the directory,
	// and local has no additions — directory should be removed entirely.
	configsDirOrg := filepath.Join(org, "configs")
	configsDirDst := filepath.Join(dst, "configs")

	assert.NoError(t, os.Mkdir(configsDirOrg, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(configsDirOrg, "base.yaml"), []byte("original"), 0644))

	assert.NoError(t, os.Mkdir(configsDirDst, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(configsDirDst, "base.yaml"), []byte("original"), 0644))

	err := RemoveStaleItems(org, src, dst, true, All)
	assert.NoError(t, err)

	// Both the file and directory should be gone.
	_, err = os.Stat(filepath.Join(configsDirDst, "base.yaml"))
	assert.True(t, os.IsNotExist(err), "stale file should be removed")
	_, err = os.Stat(configsDirDst)
	assert.True(t, os.IsNotExist(err), "empty stale directory should be removed")
}

func TestRemoveStaleItems_NestedDirsWithLocalFile(t *testing.T) {
	org := t.TempDir()
	src := t.TempDir()
	dst := t.TempDir()

	// Origin has configs/nested/base.yaml and configs/top.yaml.
	// Upstream (src) deletes everything.
	// Local added configs/nested/custom.yaml.
	assert.NoError(t, os.MkdirAll(filepath.Join(org, "configs", "nested"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(org, "configs", "top.yaml"), []byte("orig"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(org, "configs", "nested", "base.yaml"), []byte("orig"), 0644))

	assert.NoError(t, os.MkdirAll(filepath.Join(dst, "configs", "nested"), 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(dst, "configs", "top.yaml"), []byte("orig"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(dst, "configs", "nested", "base.yaml"), []byte("orig"), 0644))
	assert.NoError(t, os.WriteFile(filepath.Join(dst, "configs", "nested", "custom.yaml"), []byte("local"), 0644))

	err := RemoveStaleItems(org, src, dst, true, All)
	assert.NoError(t, err)

	// Stale files should be removed.
	_, err = os.Stat(filepath.Join(dst, "configs", "top.yaml"))
	assert.True(t, os.IsNotExist(err), "stale file top.yaml should be removed")
	_, err = os.Stat(filepath.Join(dst, "configs", "nested", "base.yaml"))
	assert.True(t, os.IsNotExist(err), "stale file base.yaml should be removed")

	// Locally-added file should be preserved.
	_, err = os.Stat(filepath.Join(dst, "configs", "nested", "custom.yaml"))
	assert.NoError(t, err, "locally-added file custom.yaml should be preserved")

	// Both parent directories should be preserved because nested/ still has content.
	_, err = os.Stat(filepath.Join(dst, "configs", "nested"))
	assert.NoError(t, err, "nested dir with local files should be preserved")
	_, err = os.Stat(filepath.Join(dst, "configs"))
	assert.NoError(t, err, "parent dir should be preserved when child dir has content")
}
