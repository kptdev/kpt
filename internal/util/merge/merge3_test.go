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

package merge

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/copyutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestMerge3_Nested_packages(t *testing.T) {
	annotationSetter := yaml.SetAnnotation("foo", "bar")
	labelSetter := yaml.SetLabel("bar", "foo")

	testCases := []struct {
		name               string
		includeSubPackages bool
		original           *pkgbuilder.Pkg
		upstream           *pkgbuilder.Pkg
		local              *pkgbuilder.Pkg
		expected           *pkgbuilder.Pkg
	}{
		{
			name:               "subpackages are merged if included",
			includeSubPackages: true,
			original:           createPkg(),
			upstream:           createPkg(annotationSetter),
			local:              createPkg(labelSetter),
			expected:           createPkg(labelSetter, annotationSetter),
		},
		{
			name:               "subpackages are not merged if not included",
			includeSubPackages: false,
			original:           createPkg(),
			upstream:           createPkg(annotationSetter),
			local:              createPkg(labelSetter),
			expected: createPkgMultipleMutators(
				[]yaml.Filter{
					labelSetter,
					annotationSetter,
				},
				[]yaml.Filter{
					labelSetter,
				},
			),
		},
		{
			name:               "local copy defines the package boundaries if different from upstream",
			includeSubPackages: false,
			original: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
		{
			name:               "upstream changes not included if in a different package",
			includeSubPackages: false,
			original: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a"). // No Kptfile
									WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewPackage("base").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewPackage("a").
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			original := pkgbuilder.ExpandPkg(t, test.original)
			updated := pkgbuilder.ExpandPkg(t, test.upstream)
			local := pkgbuilder.ExpandPkg(t, test.local)
			expected := pkgbuilder.ExpandPkg(t, test.expected)
			err := Merge3{
				OriginalPath:       original,
				UpdatedPath:        updated,
				DestPath:           local,
				MergeOnPath:        true,
				IncludeSubPackages: test.includeSubPackages,
			}.Merge()
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			diffs, err := copyutil.Diff(local, expected)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			if !assert.Empty(t, diffs.List()) {
				t.FailNow()
			}
		})
	}
}

func createPkg(mutators ...yaml.Filter) *pkgbuilder.Pkg {
	return createPkgMultipleMutators(mutators, mutators)
}

func createPkgMultipleMutators(packageMutators, subPackageMutators []yaml.Filter) *pkgbuilder.Pkg {
	return pkgbuilder.NewPackage("base").
		WithKptfile().
		WithResource(pkgbuilder.DeploymentResource, packageMutators...).
		WithSubPackages(
			pkgbuilder.NewPackage("a").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
			pkgbuilder.NewPackage("b").
				WithResource(pkgbuilder.DeploymentResource, packageMutators...).
				WithSubPackages(
					pkgbuilder.NewPackage("c").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
				),
		)
}
