// Copyright 2021 The kpt Authors
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

package pkgutil_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/testutil"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
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
			if err := pkgutil.WalkPackage(pkgPath, func(s string, info os.FileInfo, err error) error {
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
		subpackageMatcher pkg.SubpackageMatcher
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
			subpackageMatcher: pkg.Local,
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
			subpackageMatcher: pkg.None,
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
			subpackageMatcher: pkg.None,
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
			subpackageMatcher: pkg.All,
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
			subpackageMatcher: pkg.Local,
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
			subpackageMatcher: pkg.Remote,
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
			subpackageMatcher: pkg.Local,
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
			subpackageMatcher: pkg.Local,
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
			subpackageMatcher: pkg.Remote,
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
			subpackageMatcher: pkg.Remote,
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

			err := pkgutil.CopyPackage(pkgPath, dest, tc.copyRootKptfile, tc.subpackageMatcher)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			var visited []string
			if err = filepath.Walk(dest, func(s string, info os.FileInfo, err error) error {
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
									WithUpstream("github.com/GoogleContainerTools/kpt",
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
									WithUpstream("github.com/GoogleContainerTools/kpt",
										"/", "main", string(kptfilev1.ResourceMerge)),
							).
							WithResource(pkgbuilder.ConfigMapResource).
							WithSubPackages(
								pkgbuilder.NewSubPkg("bar").
									WithSubPackages(
										pkgbuilder.NewSubPkg("zork").
											WithKptfile(
												pkgbuilder.NewKptfile().
													WithUpstream("github.com/GoogleContainerTools/kpt",
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
									WithUpstream("github.com/GoogleContainerTools/kpt",
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

			paths, err := pkgutil.FindSubpackagesForPaths(pkg.Local, true, pkgPaths...)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			sort.Strings(paths)

			assert.Equal(t, tc.expected, paths)
		})
	}
}
