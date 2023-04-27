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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	v1alpha2 "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
)

func TestValidateConvert(t *testing.T) {
	testCases := map[string]struct {
		v1        PackageVariantSet
		v2        v1alpha2.PackageVariantSet
		v1tov2Err string
		v2tov1Err string
	}{
		"empty": {
			v1:        PackageVariantSet{},
			v2:        v1alpha2.PackageVariantSet{},
			v1tov2Err: "",
			v2tov1Err: "",
		},
		"empty spec": {
			v1: PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			v2: v1alpha2.PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
		},
		"upstream": {
			v1: PackageVariantSet{
				Spec: PackageVariantSetSpec{
					Upstream: &Upstream{
						Package: &Package{
							Repo: "foobar",
							Name: "barfoo",
						},
						Revision: "v8",
					},
				},
			},
			v2: v1alpha2.PackageVariantSet{
				Spec: v1alpha2.PackageVariantSetSpec{
					Upstream: &pkgvarapi.Upstream{
						Repo:     "foobar",
						Package:  "barfoo",
						Revision: "v8",
					},
				},
			},
		},
		"repositories targets": {
			v1: PackageVariantSet{
				Spec: PackageVariantSetSpec{
					Targets: []Target{
						{
							Package: &Package{
								Repo: "myrepo",
								Name: "pkg1",
							},
						},
						{
							Package: &Package{
								Repo: "myrepo",
								Name: "pkg2",
							},
						},
					},
				},
			},
			v2: v1alpha2.PackageVariantSet{
				Spec: v1alpha2.PackageVariantSetSpec{
					Targets: []v1alpha2.Target{
						{
							Repositories: []v1alpha2.RepositoryTarget{
								{
									Name:         "myrepo",
									PackageNames: []string{"pkg1", "pkg2"},
								},
							},
						},
					},
				},
			},
		},
		"repository selector target": {
			v1: PackageVariantSet{
				Spec: PackageVariantSetSpec{
					Targets: []Target{
						{
							Repositories: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"foo": "bar",
								},
							},
						},
					},
				},
			},
			v2: v1alpha2.PackageVariantSet{
				Spec: v1alpha2.PackageVariantSetSpec{
					Targets: []v1alpha2.Target{
						{
							RepositorySelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"foo": "bar",
								},
							},
						},
					},
				},
			},
		},
		"object selector target": {
			v1: PackageVariantSet{
				Spec: PackageVariantSetSpec{
					Targets: []Target{
						{
							Objects: &ObjectSelector{
								Selectors: []Selector{
									{
										Labels: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"foo": "bar",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			v2: v1alpha2.PackageVariantSet{
				Spec: v1alpha2.PackageVariantSetSpec{
					Targets: []v1alpha2.Target{
						{
							ObjectSelector: &v1alpha2.ObjectSelector{
								LabelSelector: metav1.LabelSelector{
									MatchLabels: map[string]string{
										"foo": "bar",
									},
								},
							},
						},
					},
				},
			},
			v1tov2Err: "conversion of object selector targets is not supported",
			v2tov1Err: "conversion of object selector targets is not supported",
		},
	}
	for tn, tc := range testCases {
		t.Run("v1->v2 "+tn, func(t *testing.T) {
			var v2 v1alpha2.PackageVariantSet
			err := ConvertV1alpha1ToV1alpha2(&tc.v1, &v2)
			if tc.v1tov2Err != "" {
				require.EqualError(t, err, tc.v1tov2Err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.v2, v2)
		})

		t.Run("v2->v1 "+tn, func(t *testing.T) {
			var v1 PackageVariantSet
			err := ConvertV1alpha2ToV1alpha1(&tc.v2, &v1)
			if tc.v2tov1Err != "" {
				require.EqualError(t, err, tc.v2tov1Err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.v1, v1)
		})
	}
}
