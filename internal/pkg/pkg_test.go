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

package pkg

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewPkg(t *testing.T) {
	// this test creates a folders with structure
	// foo
	// └── bar
	//     └── baz
	var tests = []struct {
		name        string
		workingDir  string
		inputPath   string
		displayPath string
	}{
		{
			name:        "invoked from working directory foo on path .",
			workingDir:  "foo",
			inputPath:   ".",
			displayPath: "foo",
		},
		{
			name:        "invoked from working directory foo/bar on path ../",
			workingDir:  "foo/bar",
			inputPath:   "../",
			displayPath: "foo",
		},
		{
			name:        "invoked from working directory foo on nested package baz",
			workingDir:  "foo",
			inputPath:   "./bar/baz",
			displayPath: "baz",
		},
		{
			name:        "invoked from working directory foo/bar on nested package baz",
			workingDir:  "foo/bar",
			inputPath:   "../../foo/bar/baz",
			displayPath: "baz",
		},
		{
			name:        "invoked from working directory baz on ancestor package foo",
			workingDir:  "foo/bar/baz",
			inputPath:   "../../",
			displayPath: "foo",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			assert.NoError(t, err)
			err = os.MkdirAll(filepath.Join(dir, "foo", "bar", "baz"), 0700)
			assert.NoError(t, err)
			revert := Chdir(t, filepath.Join(dir, test.workingDir))
			defer revert()
			p, err := New(test.inputPath)
			assert.NoError(t, err)
			assert.Equal(t, test.displayPath, string(p.DisplayPath))
		})
	}
}

func TestAdjustDisplayPathForSubpkg(t *testing.T) {
	// this test creates a folders with structure
	// rootPkgParentDir
	// └── rootPkg
	//     └── subPkg
	//         └── nestedPkg
	var tests = []struct {
		name                 string
		workingDir           string
		pkgPath              string
		subPkgPath           string
		rootPkgParentDirPath string
		displayPath          string
	}{
		{
			name:        "display path of subPkg should include rootPkg",
			workingDir:  "rootPkg",
			pkgPath:     ".",
			subPkgPath:  "./subPkg",
			displayPath: "rootPkg/subPkg",
		},
		{
			name:        "display path of nestedPkg should include rootPkg/subPkg",
			workingDir:  "rootPkg",
			pkgPath:     ".",
			subPkgPath:  "./subPkg/nestedPkg",
			displayPath: "rootPkg/subPkg/nestedPkg",
		},
		{
			name:        "display path of subPkg should include rootPkg independent of workingDir",
			workingDir:  "rootPkg/subPkg",
			pkgPath:     "../",
			subPkgPath:  "../subPkg",
			displayPath: "rootPkg/subPkg",
		},
		{
			name:        "display path of nestedPkg should include rootPkg independent of workingDir 1",
			workingDir:  "rootPkg/subPkg/nestedPkg",
			pkgPath:     "../../",
			subPkgPath:  "../../subPkg/nestedPkg",
			displayPath: "rootPkg/subPkg/nestedPkg",
		},
		{
			name:                 "display path of nestedPkg should include rootPkg independent of workingDir 2",
			workingDir:           "rootPkg",
			rootPkgParentDirPath: "../",
			pkgPath:              "./subPkg",
			subPkgPath:           "./subPkg/nestedPkg",
			displayPath:          "rootPkg/subPkg/nestedPkg",
		},
		{
			name:                 "display path of nestedPkg should include rootPkg independent of workingDir 3",
			workingDir:           "rootPkg/subPkg",
			rootPkgParentDirPath: "../../",
			pkgPath:              "../subPkg",
			subPkgPath:           "../subPkg/nestedPkg",
			displayPath:          "rootPkg/subPkg/nestedPkg",
		},
		{
			name:                 "display path of nestedPkg should include rootPkg independent of workingDir 4",
			workingDir:           "rootPkg/subPkg/nestedPkg",
			rootPkgParentDirPath: "../../../",
			pkgPath:              "../../subPkg",
			subPkgPath:           "../../subPkg/nestedPkg",
			displayPath:          "rootPkg/subPkg/nestedPkg",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "")
			defer os.RemoveAll(dir)
			assert.NoError(t, err)
			err = os.MkdirAll(filepath.Join(dir, "rootPkgParentDir", "rootPkg", "subPkg", "nestedPkg"), 0700)
			assert.NoError(t, err)
			revert := Chdir(t, filepath.Join(dir, "rootPkgParentDir", test.workingDir))
			defer revert()
			parent, err := New(test.pkgPath)
			assert.NoError(t, err)
			if test.rootPkgParentDirPath != "" {
				rootPkg, err := New(test.rootPkgParentDirPath)
				assert.NoError(t, err)
				parent.rootPkgParentDirPath = string(rootPkg.UniquePath)
			}
			subPkg, err := New(test.subPkgPath)
			assert.NoError(t, err)
			err = parent.adjustDisplayPathForSubpkg(subPkg)
			assert.NoError(t, err)
			assert.Equal(t, test.displayPath, string(subPkg.DisplayPath))
		})
	}
}

func TestFilterMetaResources(t *testing.T) {
	tests := map[string]struct {
		resources []string
		expected  []string
	}{
		"no resources": {
			resources: nil,
			expected:  nil,
		},

		"nothing to filter": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},

		"filter out metadata": {
			resources: []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}`,
				`
apiVersion: config.kpt.dev/v1
Kind: FunctionPermission
Metadata:
  Name: functionPermission
Spec:
  Allow:
  - imageName: gcr.io/my-project/*…..
  Permissions:
  - network
  - mount
  Disallow:
  - Name: gcr.io/my-project/*`,
				`
apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3`,
				`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: mysql
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:v0.1
      configMap:
        replicas: 5
`,
				`
apiVersion: kpt.dev/v1
kind: Pipeline
sources:
  - "."`,
			},
			expected: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # {"$kpt-set":"replicas"}
`,
				`apiVersion: custom.io/v1
kind: Custom
spec:
  image: nginx:1.2.3
`,
			},
		},
	}

	for name := range tests {
		test := tests[name]
		t.Run(name, func(t *testing.T) {
			var nodes []*yaml.RNode

			for _, r := range test.resources {
				res, err := yaml.Parse(r)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				nodes = append(nodes, res)
			}

			filteredRes, err := filterMetaResources(nodes, nil)
			if err != nil {
				t.Errorf("unexpected error in filtering meta resources: %v", err)
			}
			if len(filteredRes) != len(test.expected) {
				t.Fatal("length of filtered resources not equal to expected")
			}

			for i, r := range filteredRes {
				res, err := r.String()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, test.expected[i], res)
			}
		})
	}
}

func TestDirectSubpackages(t *testing.T) {
	testCases := map[string]struct {
		pkg      *pkgbuilder.RootPkg
		expected []string
	}{
		"includes remote subpackages": {
			pkg: pkgbuilder.NewRootPkg().
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
			expected: []string{
				"foo",
			},
		},
		"includes local subpackages": {
			pkg: pkgbuilder.NewRootPkg().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("foo").
						WithKptfile().
						WithResource(pkgbuilder.ConfigMapResource),
				),
			expected: []string{
				"foo",
			},
		},
		"does not include root package": {
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			expected: []string{},
		},
		"does not include nested remote subpackages": {
			pkg: pkgbuilder.NewRootPkg().
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
			expected: []string{
				"foo",
			},
		},
		"does not include nested local subpackages": {
			pkg: pkgbuilder.NewRootPkg().
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
			expected: []string{
				"foo",
				"subpkg",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, nil)
			defer os.RemoveAll(pkgPath)
			p, err := New(pkgPath)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			subPkgs, err := p.DirectSubpackages()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			relPaths := []string{}
			for _, subPkg := range subPkgs {
				fullPath := subPkg.UniquePath.String()
				relPath, err := filepath.Rel(pkgPath, fullPath)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				relPaths = append(relPaths, relPath)
			}
			sort.Strings(relPaths)

			assert.Equal(t, tc.expected, relPaths)
		})
	}
}

//nolint:scopelint
func TestSubpackages(t *testing.T) {
	type variants struct {
		matcher   []SubpackageMatcher
		recursive []bool
		expected  []string
	}

	testCases := map[string]struct {
		pkg   *pkgbuilder.RootPkg
		cases []variants
	}{
		"remote and local nested subpackages": {
			// root
			//  ├── remote-sub1 (remote)
			//  │   ├── Kptfile
			//  │   └── directory
			//  │       └── remote-sub3 (remote)
			//  │           └── Kptfile
			//  └── local-sub1 (local)
			//      ├── Kptfile
			//      ├── directory
			//      │   └── remote-sub3 (remote)
			//      │       └── Kptfile
			//      └── local-sub2 (local)
			//          └── Kptfile
			pkg: pkgbuilder.NewRootPkg().
				WithSubPackages(
					pkgbuilder.NewSubPkg("remote-sub1").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt",
									"/", "main", string(kptfilev1.ResourceMerge)),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("directory").
								WithSubPackages(
									pkgbuilder.NewSubPkg("remote-sub3").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1.ResourceMerge)),
										),
								),
						),
					pkgbuilder.NewSubPkg("local-sub1").
						WithKptfile().
						WithSubPackages(
							pkgbuilder.NewSubPkg("directory").
								WithSubPackages(
									pkgbuilder.NewSubPkg("remote-sub3").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1.ResourceMerge)),
										),
								),
							pkgbuilder.NewSubPkg("local-sub2").
								WithKptfile(),
						),
				),
			cases: []variants{
				{
					matcher:   []SubpackageMatcher{All},
					recursive: []bool{true},
					expected: []string{
						"local-sub1",
						"local-sub1/directory/remote-sub3",
						"local-sub1/local-sub2",
						"remote-sub1",
						"remote-sub1/directory/remote-sub3",
					},
				},
				{
					matcher:   []SubpackageMatcher{All},
					recursive: []bool{false},
					expected: []string{
						"local-sub1",
						"remote-sub1",
					},
				},
				{
					matcher:   []SubpackageMatcher{Remote},
					recursive: []bool{true},
					expected: []string{
						"local-sub1/directory/remote-sub3",
						"remote-sub1",
						"remote-sub1/directory/remote-sub3",
					},
				},
				{
					matcher:   []SubpackageMatcher{Remote},
					recursive: []bool{false},
					expected: []string{
						"remote-sub1",
					},
				},
				{
					matcher:   []SubpackageMatcher{Local},
					recursive: []bool{true},
					expected: []string{
						"local-sub1",
						"local-sub1/local-sub2",
					},
				},
				{
					matcher:   []SubpackageMatcher{Local},
					recursive: []bool{false},
					expected: []string{
						"local-sub1",
					},
				},
				{
					matcher:   []SubpackageMatcher{None},
					recursive: []bool{false},
					expected:  []string{},
				},
				{
					matcher:   []SubpackageMatcher{None},
					recursive: []bool{true},
					expected:  []string{},
				},
			},
		},
		"no subpackages": {
			// root
			//  └── Kptfile
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(),
			cases: []variants{
				{
					matcher:   []SubpackageMatcher{All, Local, Remote, None},
					recursive: []bool{true, false},
					expected:  []string{},
				},
			},
		},
		"no Kptfile in root": {
			// root
			pkg: pkgbuilder.NewRootPkg(),
			cases: []variants{
				{
					matcher:   []SubpackageMatcher{All, Local, Remote, None},
					recursive: []bool{true, false},
					expected:  []string{},
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(pkgPath)
			}()

			for _, v := range tc.cases {
				for _, matcher := range v.matcher {
					for _, recursive := range v.recursive {
						t.Run(fmt.Sprintf("matcher:%s-recursive:%t", matcher, recursive), func(t *testing.T) {
							paths, err := Subpackages(pkgPath, matcher, recursive)
							if !assert.NoError(t, err) {
								t.FailNow()
							}

							sort.Strings(paths)
							sort.Strings(v.expected)

							assert.Equal(t, v.expected, paths)
						})
					}
				}
			}
		})
	}
}

func TestSubpackages_symlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	pkg := pkgbuilder.NewRootPkg().
		WithResource(pkgbuilder.DeploymentResource).
		WithSubPackages(
			pkgbuilder.NewSubPkg("subpkg").
				WithKptfile().
				WithResource(pkgbuilder.ConfigMapResource),
		)

	pkgPath := pkg.ExpandPkg(t, nil)
	defer func() {
		_ = os.RemoveAll(pkgPath)
	}()

	symLinkOld := filepath.Join(pkgPath, "subpkg")
	symLinkNew := filepath.Join(pkgPath, "symlink-subpkg")

	err := os.Symlink(symLinkOld, symLinkNew)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	paths, err := Subpackages(pkgPath, All, true)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, []string{"subpkg"}, paths)
}

func TestFunctionConfigFilePaths(t *testing.T) {
	type variants struct {
		recursive bool
		expected  []string
	}

	testCases := map[string]struct {
		pkg   *pkgbuilder.RootPkg
		cases []variants
	}{
		"remote and local nested subpackages": {
			// root
			//  ├── remote-sub1 (remote)
			//  │   ├── Kptfile
			//  │   ├── fn-config.yaml
			//  │   └── directory
			//  │       └── remote-sub3 (remote)
			//  │           ├── Kptfile
			//  │           ├── fn-config2.yaml
			//  │           └── fn-config1.yaml
			//  └── local-sub1 (local)
			//      ├── Kptfile
			//      ├── fn-config.yaml
			//      ├── directory
			//      │   └── remote-sub3 (remote)
			//      │       └── Kptfile
			//      └── local-sub2 (local)
			//          ├── fn-config.yaml
			//          └── Kptfile
			pkg: pkgbuilder.NewRootPkg().
				WithSubPackages(
					pkgbuilder.NewSubPkg("remote-sub1").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithUpstream("github.com/GoogleContainerTools/kpt",
									"/", "main", string(kptfilev1.ResourceMerge)).
								WithPipeline(
									pkgbuilder.NewFunction("image").
										WithConfigPath("fn-config.yaml"),
								),
						).
						WithFile("fn-config.yaml", "I'm function config file.").
						WithSubPackages(
							pkgbuilder.NewSubPkg("directory").
								WithSubPackages(
									pkgbuilder.NewSubPkg("remote-sub3").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1.ResourceMerge)).
												WithPipeline(
													pkgbuilder.NewFunction("image").
														WithConfigPath("fn-config1.yaml"),
													pkgbuilder.NewFunction("image").
														WithConfigPath("fn-config2.yaml"),
												),
										).
										WithFile("fn-config1.yaml", "I'm function config file.").
										WithFile("fn-config2.yaml", "I'm function config file."),
								),
						),
					pkgbuilder.NewSubPkg("local-sub1").
						WithKptfile(
							pkgbuilder.NewKptfile().
								WithPipeline(pkgbuilder.NewFunction("image").
									WithConfigPath("fn-config.yaml")),
						).
						WithFile("fn-config.yaml", "I'm function config file.").
						WithSubPackages(
							pkgbuilder.NewSubPkg("directory").
								WithSubPackages(
									pkgbuilder.NewSubPkg("remote-sub3").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1.ResourceMerge)),
										),
								),
							pkgbuilder.NewSubPkg("local-sub2").
								WithKptfile(
									pkgbuilder.NewKptfile().
										WithPipeline(pkgbuilder.NewFunction("image").
											WithConfigPath("fn-config.yaml")),
								).
								WithFile("fn-config.yaml", "I'm function config file."),
						),
				),
			cases: []variants{
				{
					recursive: true,
					expected: []string{
						"local-sub1/fn-config.yaml",
						"local-sub1/local-sub2/fn-config.yaml",
						"remote-sub1/directory/remote-sub3/fn-config1.yaml",
						"remote-sub1/directory/remote-sub3/fn-config2.yaml",
						"remote-sub1/fn-config.yaml",
					},
				},
				{
					recursive: false,
					expected:  nil,
				},
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pkgPath := tc.pkg.ExpandPkg(t, nil)
			defer func() {
				_ = os.RemoveAll(pkgPath)
			}()

			for _, v := range tc.cases {
				v := v
				t.Run(fmt.Sprintf("recursive:%t", v.recursive), func(t *testing.T) {
					paths, err := FunctionConfigFilePaths(types.UniquePath(pkgPath), v.recursive)
					if !assert.NoError(t, err) {
						t.FailNow()
					}

					pathsList := paths.List()
					sort.Strings(pathsList)
					sort.Strings(v.expected)

					assert.Equal(t, v.expected, pathsList)
				})
			}
		})
	}
}

func Chdir(t *testing.T, path string) func() {
	cwd, err := os.Getwd()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	revertFunc := func() {
		if err := os.Chdir(cwd); err != nil {
			panic(err)
		}
	}
	err = os.Chdir(path)
	if !assert.NoError(t, err) {
		defer revertFunc()
		t.FailNow()
	}
	return revertFunc
}
