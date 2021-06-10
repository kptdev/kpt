// Copyright 2021 Google LLC
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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	. "github.com/GoogleContainerTools/kpt/internal/pkg"
	testing2 "github.com/GoogleContainerTools/kpt/internal/pkg/testing"
	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
)

func TestPackageWalker_Walk_PackageDoesNotExist(t *testing.T) {
	dir, err := ioutil.TempDir("", "kpt-test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	p := testing2.CreatePkgOrFail(t, filepath.Join(dir, "doesNotExist"))

	err = (&Walker{
		FileMatcher: AllMatcher,
	}).Walk(p, func(s string, info os.FileInfo, err error) error {
		return err
	})
	assert.Error(t, err)
}

func TestPackageWalker_Walk(t *testing.T) {
	testCases := map[string]struct {
		pkg                         *pkgbuilder.RootPkg
		fileMatcher                 FileMatcher
		ignoreKptfileIgnorePatterns bool
		expectedPaths               []string
	}{
		"empty package without Kptfile": {
			pkg:                         pkgbuilder.NewRootPkg(),
			fileMatcher:                 AllMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
			},
		},
		"empty package with Kptfile but no ignore list": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				),
			fileMatcher:                 AllMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
				kptfilev1alpha2.KptFileName,
			},
		},
		"use file matcher to only get yaml files": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithResource(pkgbuilder.DeploymentResource).
				WithFile("foo.txt", "this is a test"),
			fileMatcher:                 YamlMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
				"deployment.yaml",
			},
		},
		"subpackages are skipped": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile(),
				).
				WithFile("deployment.yml", "yaml").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpkg").
						WithKptfile().
						WithResource(pkgbuilder.SecretResource),
				),
			fileMatcher:                 KptfileYamlMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
				"Kptfile",
				"deployment.yml",
			},
		},
		"files covered by the ignore list are not included": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore("deployment.yml"),
				).
				WithFile("deployment.yml", "yaml"),
			fileMatcher:                 KptfileYamlMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
				"Kptfile",
			},
		},
		"directories covered by the ignore list are not included": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore("subdir"),
				).
				WithFile("deployment.yml", "yaml").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithResource(pkgbuilder.SecretResource),
				),
			fileMatcher:                 KptfileYamlMatcher,
			ignoreKptfileIgnorePatterns: false,
			expectedPaths: []string{
				".",
				"Kptfile",
				"deployment.yml",
			},
		},
		"ignore list are disregarded if ignoreKptfileIgnorePatterns is false": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore("subdir"),
				).
				WithFile("deployment.yml", "yaml").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithResource(pkgbuilder.SecretResource),
				),
			fileMatcher:                 KptfileYamlMatcher,
			ignoreKptfileIgnorePatterns: true,
			expectedPaths: []string{
				".",
				"Kptfile",
				"deployment.yml",
				"subdir",
				"subdir/secret.yaml",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(pkgPath)
			}()
			p := testing2.CreatePkgOrFail(t, pkgPath)

			cbPaths := make([]string, 0)
			err := (&Walker{
				FileMatcher:                 tc.fileMatcher,
				IgnoreKptfileIgnorePatterns: tc.ignoreKptfileIgnorePatterns,
			}).Walk(p, func(s string, info os.FileInfo, err error) error {
				cbPaths = append(cbPaths, s)
				return err
			})
			assert.NoError(t, err)

			relCbPaths := toRelPath(t, pkgPath, cbPaths)
			sort.Strings(tc.expectedPaths)
			sort.Strings(relCbPaths)
			assert.Equal(t, tc.expectedPaths, relCbPaths)
		})
	}
}

func TestPackageWalker_Walk_Ignore(t *testing.T) {
	testCases := map[string]struct {
		pkg           *pkgbuilder.RootPkg
		expectedPaths []string
	}{
		"absolute path for file": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"/foo.txt",
						),
				).
				WithFile("foo.txt", "foo").
				WithFile("bar.txt", "bar").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"bar.txt",
				"subdir",
				"subdir/foo.txt",
			},
		},
		"absolute path for directory": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"/subdir",
						),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
					pkgbuilder.NewSubPkg("othersubdir").
						WithSubPackages(
							pkgbuilder.NewSubPkg("subdir").
								WithFile("foo.txt", "foo"),
						),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"othersubdir",
				"othersubdir/subdir",
				"othersubdir/subdir/foo.txt",
			},
		},
		"relative path for file": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"foo.txt",
						),
				).
				WithFile("foo.txt", "foo").
				WithFile("bar.txt", "bar").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"bar.txt",
				"subdir",
			},
		},
		"relative path for directory": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"subdir",
						),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
					pkgbuilder.NewSubPkg("othersubdir").
						WithSubPackages(
							pkgbuilder.NewSubPkg("subdir").
								WithFile("foo.txt", "foo"),
						),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"othersubdir",
			},
		},
		"accept pattern for file": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"*.txt",
							"!/subdir/foo.txt",
						),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"subdir",
				"subdir/foo.txt",
			},
		},
		"accept pattern for file inside excluded directory": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"subdir",
							"!/subdir/foo.txt",
						),
				).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithFile("foo.txt", "foo"),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
			},
		},
		"directory pattern": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"foo/",
						),
				).
				WithFile("foo", "foo").
				WithSubPackages(
					pkgbuilder.NewSubPkg("subdir").
						WithSubPackages(
							pkgbuilder.NewSubPkg("foo").
								WithFile("bar.txt", "bar"),
						),
				),
			expectedPaths: []string{
				".",
				"Kptfile",
				"foo",
				"subdir",
			},
		},
		"glob pattern": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(
					pkgbuilder.NewKptfile().
						WithIgnore(
							"*.txt",
						),
				).
				WithFile("foo", "foo").
				WithFile("bar.txt", "bar").
				WithFile("other.txt", "other"),
			expectedPaths: []string{
				".",
				"Kptfile",
				"foo",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(pkgPath)
			}()
			p := testing2.CreatePkgOrFail(t, pkgPath)

			cbPaths := make([]string, 0)
			err := (&Walker{
				FileMatcher: AllMatcher,
			}).Walk(p, func(s string, info os.FileInfo, err error) error {
				cbPaths = append(cbPaths, s)
				return err
			})
			assert.NoError(t, err)

			relCbPaths := toRelPath(t, pkgPath, cbPaths)
			sort.Strings(tc.expectedPaths)
			sort.Strings(relCbPaths)
			assert.Equal(t, tc.expectedPaths, relCbPaths)
		})
	}
}

func toRelPath(t *testing.T, base string, paths []string) []string {
	relPaths := make([]string, 0)
	for _, p := range paths {
		rel, err := filepath.Rel(base, p)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		relPaths = append(relPaths, rel)
	}
	return relPaths
}
