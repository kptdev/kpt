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

	"github.com/GoogleContainerTools/kpt/internal/testutil"
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
		original           *pkgbuilder.RootPkg
		upstream           *pkgbuilder.RootPkg
		local              *pkgbuilder.RootPkg
		expected           *pkgbuilder.RootPkg
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
			original: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
		{
			name:               "upstream changes not included if in a different package",
			includeSubPackages: false,
			original: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource),
				),
			upstream: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, annotationSetter),
				),
			local: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a"). // No Kptfile
									WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
			expected: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, labelSetter, annotationSetter).
				WithSubPackages(
					pkgbuilder.NewSubPkg("a").
						WithResource(pkgbuilder.DeploymentResource, labelSetter),
				),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			original := test.original.ExpandPkg(t, testutil.EmptyReposInfo)
			updated := test.upstream.ExpandPkg(t, testutil.EmptyReposInfo)
			local := test.local.ExpandPkg(t, testutil.EmptyReposInfo)
			expected := test.expected.ExpandPkg(t, testutil.EmptyReposInfo)
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

func createPkg(mutators ...yaml.Filter) *pkgbuilder.RootPkg {
	return createPkgMultipleMutators(mutators, mutators)
}

func createPkgMultipleMutators(packageMutators, subPackageMutators []yaml.Filter) *pkgbuilder.RootPkg {
	return pkgbuilder.NewRootPkg().
		WithKptfile().
		WithResource(pkgbuilder.DeploymentResource, packageMutators...).
		WithSubPackages(
			pkgbuilder.NewSubPkg("a").
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
			pkgbuilder.NewSubPkg("b").
				WithResource(pkgbuilder.DeploymentResource, packageMutators...).
				WithSubPackages(
					pkgbuilder.NewSubPkg("c").
						WithKptfile().
						WithResource(pkgbuilder.DeploymentResource, subPackageMutators...),
				),
		)
}
