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
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/GoogleContainerTools/kpt/internal/types"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestNewPkg(t *testing.T) {
	var tests = []struct {
		name        string
		inputPath   string
		uniquePath  string
		displayPath string
	}{
		{
			name:        "test1",
			inputPath:   ".",
			displayPath: ".",
		},
		{
			name:        "test2",
			inputPath:   "../",
			displayPath: "..",
		},
		{
			name:        "test3",
			inputPath:   "./foo/bar/",
			displayPath: "foo/bar",
		},
	}
	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			p, err := New(test.inputPath)
			assert.NoError(t, err)
			assert.Equal(t, test.displayPath, string(p.DisplayPath))
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
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
metadata:
  name: mysql
setterDefinitions:
  replicas:
    description: "replica setter"
    type: integer
setterValues:
  replicas: 5`,
				`
apiVersion: kpt.dev/v1alpha1
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
									"/", "main", string(kptfilev1alpha2.ResourceMerge)),
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
									"/", "main", string(kptfilev1alpha2.ResourceMerge)),
						).
						WithResource(pkgbuilder.ConfigMapResource).
						WithSubPackages(
							pkgbuilder.NewSubPkg("bar").
								WithSubPackages(
									pkgbuilder.NewSubPkg("zork").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1alpha2.ResourceMerge)),
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
									"/", "main", string(kptfilev1alpha2.ResourceMerge)),
						).
						WithSubPackages(
							pkgbuilder.NewSubPkg("directory").
								WithSubPackages(
									pkgbuilder.NewSubPkg("remote-sub3").
										WithKptfile(
											pkgbuilder.NewKptfile().
												WithUpstream("github.com/GoogleContainerTools/kpt",
													"/", "main", string(kptfilev1alpha2.ResourceMerge)),
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
													"/", "main", string(kptfilev1alpha2.ResourceMerge)),
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
			},
		},
		"no subpackages": {
			// root
			//  └── Kptfile
			pkg: pkgbuilder.NewRootPkg().
				WithKptfile(),
			cases: []variants{
				{
					matcher:   []SubpackageMatcher{All, Local, Remote},
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
					matcher:   []SubpackageMatcher{All, Local, Remote},
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
									"/", "main", string(kptfilev1alpha2.ResourceMerge)).
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
													"/", "main", string(kptfilev1alpha2.ResourceMerge)).
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
													"/", "main", string(kptfilev1alpha2.ResourceMerge)),
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
