// Copyright 2023 The kpt Authors
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

package packagevariantset

import (
	"context"
	"testing"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

func TestRenderPackageVariantSpec(t *testing.T) {
	var repoList configapi.RepositoryList
	require.NoError(t, yaml.Unmarshal([]byte(`
apiVersion: config.porch.kpt.dev/v1alpha1
kind: RepositoryList
metadata:
  name: my-repo-list
items:
- apiVersion: config.porch.kpt.dev/v1alpha1
  kind: Repository
  metadata:
    name: my-repo-1
    labels:
      foo: bar
      abc: def
- apiVersion: config.porch.kpt.dev/v1alpha1
  kind: Repository
  metadata:
    name: my-repo-2
    labels:
      foo: bar
      abc: def
      efg: hij
`), &repoList))

	adoptExisting := pkgvarapi.AdoptionPolicyAdoptExisting
	deletionPolicyDelete := pkgvarapi.DeletionPolicyDelete
	pvs := api.PackageVariantSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvs",
			Namespace: "default",
		},
		Spec: api.PackageVariantSetSpec{
			Upstream: &pkgvarapi.Upstream{Repo: "up-repo", Package: "up-pkg", Revision: "v2"},
		},
	}
	upstreamPR := porchapi.PackageRevision{}
	testCases := map[string]struct {
		downstream   pvContext
		expectedSpec pkgvarapi.PackageVariantSpec
		expectedErrs []string
	}{
		"no template": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "r",
					Package: "p",
				},
			},
			expectedErrs: nil,
		},
		"template downstream.repo": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
				template: &api.PackageVariantTemplate{
					Downstream: &api.DownstreamTemplate{
						Repo: pointer.String("new-r"),
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "new-r",
					Package: "p",
				},
			},
			expectedErrs: nil,
		},
		"template downstream.package": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
				template: &api.PackageVariantTemplate{
					Downstream: &api.DownstreamTemplate{
						Package: pointer.String("new-p"),
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "r",
					Package: "new-p",
				},
			},
			expectedErrs: nil,
		},
		"template adoption and deletion": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
				template: &api.PackageVariantTemplate{
					AdoptionPolicy: &adoptExisting,
					DeletionPolicy: &deletionPolicyDelete,
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "r",
					Package: "p",
				},
				AdoptionPolicy: "adoptExisting",
				DeletionPolicy: "delete",
			},
			expectedErrs: nil,
		},
		"template static labels and annotations": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
				template: &api.PackageVariantTemplate{
					Labels: map[string]string{
						"foo":   "bar",
						"hello": "there",
					},
					Annotations: map[string]string{
						"foobar": "barfoo",
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "r",
					Package: "p",
				},
				Labels: map[string]string{
					"foo":   "bar",
					"hello": "there",
				},
				Annotations: map[string]string{
					"foobar": "barfoo",
				},
			},
			expectedErrs: nil,
		},
		"template static packageContext": {
			downstream: pvContext{
				repo:        "r",
				packageName: "p",
				template: &api.PackageVariantTemplate{
					PackageContext: &api.PackageContextTemplate{
						Data: map[string]string{
							"foo":   "bar",
							"hello": "there",
						},
						RemoveKeys: []string{"foobar", "barfoo"},
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "r",
					Package: "p",
				},
				PackageContext: &pkgvarapi.PackageContext{
					Data: map[string]string{
						"foo":   "bar",
						"hello": "there",
					},
					RemoveKeys: []string{"foobar", "barfoo"},
				},
			},
			expectedErrs: nil,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			pvSpec, err := renderPackageVariantSpec(context.Background(), &pvs, &repoList, &upstreamPR, tc.downstream)
			require.NoError(t, err)
			require.Equal(t, &tc.expectedSpec, pvSpec)
		})
	}
}
