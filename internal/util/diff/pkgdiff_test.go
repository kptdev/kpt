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

package diff

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/testutil/pkgbuilder"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

func TestPkgDiff(t *testing.T) {
	testCases := []struct {
		name string
		pkg1 *pkgbuilder.RootPkg
		pkg2 *pkgbuilder.RootPkg
		diff sets.String
	}{
		{
			name: "equal packages doesn't have a diff",
			pkg1: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile()).
				WithResource(pkgbuilder.DeploymentResource),
			pkg2: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile()).
				WithResource(pkgbuilder.DeploymentResource),
			diff: toStringSet(),
		},
		{
			name: "different files between packages",
			pkg1: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource),
			pkg2: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.ConfigMapResource),
			diff: toStringSet("configmap.yaml", "deployment.yaml"),
		},
		{
			name: "different upstream in Kptfile is not a diff",
			pkg1: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstream("github.com/GoogleContainerTools/kpt", "/", "master")).
				WithResource(pkgbuilder.DeploymentResource),
			pkg2: pkgbuilder.NewRootPkg().
				WithKptfile(pkgbuilder.NewKptfile().
					WithUpstream("github.com/GoogleContainerTools/kpt", "/", "kpt/v1")).
				WithResource(pkgbuilder.DeploymentResource),
			diff: toStringSet(),
		},
		{
			name: "subpackages are not included",
			pkg1: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpackage").
						WithKptfile(pkgbuilder.NewKptfile()).
						WithResource(pkgbuilder.DeploymentResource),
				),
			pkg2: pkgbuilder.NewRootPkg().
				WithKptfile().
				WithResource(pkgbuilder.DeploymentResource).
				WithSubPackages(
					pkgbuilder.NewSubPkg("subpackage").
						WithKptfile(pkgbuilder.NewKptfile().
							WithSetters(
								pkgbuilder.NewSetter("bar", "foo"),
							),
						).
						WithResource(pkgbuilder.ConfigMapResource),
				),
			diff: toStringSet(),
		},
	}

	for i := range testCases {
		test := testCases[i]
		t.Run(test.name, func(t *testing.T) {
			pkg1Dir := pkgbuilder.ExpandPkg(t, test.pkg1, map[string]string{})
			pkg2Dir := pkgbuilder.ExpandPkg(t, test.pkg2, map[string]string{})
			diff, err := PkgDiff(pkg1Dir, pkg2Dir)
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			assert.Equal(t, 0, len(diff.SymmetricDifference(test.diff)))
		})
	}
}

func toStringSet(files ...string) sets.String {
	s := sets.String{}
	for _, f := range files {
		s.Insert(f)
	}
	return s
}
