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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"
)

func TestValidatePackageVariantSet(t *testing.T) {
	packageVariantHeader := `apiVersion: config.porch.kpt.dev
kind: PackageVariantSet
metadata:
  name: my-pv`

	testCases := map[string]struct {
		packageVariant string
		expectedErrs   []string
	}{
		"empty spec": {
			packageVariant: packageVariantHeader,
			expectedErrs: []string{"spec.upstream is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream package": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    repo: foo
    revision: v1`,
			expectedErrs: []string{"spec.upstream.package is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream repo": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package: foopkg
    revision: v3`,
			expectedErrs: []string{"spec.upstream.repo is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream revision": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    repo: foo
    package: foopkg`,
			expectedErrs: []string{"spec.upstream.revision is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"invalid targets": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - repositories:
    - name: ""
  - repositories:
    - name: bar
    repositorySelector:
      foo: bar
  - repositories:
    - name: bar
      packageNames:
      - ""
      - foo
      `,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0].repositories[0].name cannot be empty",
				"spec.targets[1] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[2].repositories[0].packageNames[0] cannot be empty",
			},
		},
		"invalid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      adoptionPolicy: invalid
      deletionPolicy: invalid
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template.adoptionPolicy can only be \"adoptNone\" or \"adoptExisting\"",
				"spec.targets[0].template.deletionPolicy can only be \"orphan\" or \"delete\"",
			},
		},
		"valid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  adoptionPolicy: adoptExisting
  deletionPolicy: orphan
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"downstream values and expressions do not mix": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      downstream:
        repo: "foo"
        repoExpr: "'bar'"
        package: "p"
        packageExpr: "'p'"
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template may specify only one of `downstream.repo` and `downstream.repoExpr`",
				"spec.targets[0].template may specify only one of `downstream.package` and `downstream.packageExpr`",
			},
		},
		"MapExprs do not allow both expr-and non-expr for same field": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - template:
      labelExprs:
      - key: "foo"
        keyExpr: "'bar'"
        value: "bar"
      - key: "foo"
        value: "bar"
        valueExpr: "'bar'"
      annotationExprs:
      - key: "foo"
        keyExpr: "'bar'"
        value: "bar"
      - key: "foo"
        value: "bar"
        valueExpr: "'bar'"
      packageContext:
        dataExprs:
          - key: "foo"
            keyExpr: "'bar'"
            value: "bar"
          - key: "foo"
            value: "bar"
            valueExpr: "'bar'"
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] must specify one of `repositories`, `repositorySelector`, or `objectSelector`",
				"spec.targets[0].template.labelExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.labelExprs[1] may specify only one of `value` and `valueExpr`",
				"spec.targets[0].template.annotationExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.annotationExprs[1] may specify only one of `value` and `valueExpr`",
				"spec.targets[0].template.packageContext.dataExprs[0] may specify only one of `key` and `keyExpr`",
				"spec.targets[0].template.packageContext.dataExprs[1] may specify only one of `value` and `valueExpr`",
			},
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var pvs api.PackageVariantSet
			require.NoError(t, yaml.Unmarshal([]byte(tc.packageVariant), &pvs))
			actualErrs := validatePackageVariantSet(&pvs)
			require.Equal(t, len(tc.expectedErrs), len(actualErrs))
			for i := range actualErrs {
				require.EqualError(t, actualErrs[i], tc.expectedErrs[i])
			}

		})
	}
}

func TestConvertObjectToRNode(t *testing.T) {
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true})
	r := PackageVariantSetReconciler{serializer: s}

	t.Run("pod", func(t *testing.T) {
		input := `apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  name: my-pod
spec:
  containers: null
status: {}
`
		var pod corev1.Pod
		require.NoError(t, yaml.Unmarshal([]byte(input), &pod))
		n, err := r.convertObjectToRNode(&pod)
		require.NoError(t, err)
		require.Equal(t, input, n.MustString())
	})

	t.Run("repository", func(t *testing.T) {
		input := `apiVersion: config.porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  creationTimestamp: null
  name: my-repo
spec: {}
status: {}
`
		var repo configapi.Repository
		require.NoError(t, yaml.Unmarshal([]byte(input), &repo))
		n, err := r.convertObjectToRNode(&repo)
		require.NoError(t, err)
		require.Equal(t, input, n.MustString())
	})
}

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

func TestEnsurePackageVariants(t *testing.T) {
	downstreams := []pvContext{
		{repo: "dn-1", packageName: "dn-1"},
		{repo: "dn-3", packageName: "dn-3"},
	}
	fc := &fakeClient{}
	reconciler := &PackageVariantSetReconciler{Client: fc}
	require.NoError(t, reconciler.ensurePackageVariants(context.Background(),
		&api.PackageVariantSet{
			ObjectMeta: metav1.ObjectMeta{Name: "my-pvs"},
			Spec: api.PackageVariantSetSpec{
				Upstream: &pkgvarapi.Upstream{Repo: "up", Package: "up", Revision: "up"},
			},
		},
		&configapi.RepositoryList{},
		&porchapi.PackageRevision{},
		downstreams))
	require.Equal(t, 2, len(fc.objects))
	require.Equal(t, "my-pv-1", fc.objects[0].GetName())
	require.Equal(t, "my-pvs-8680372821ea923a2c068ad9fa32ffd876e9fb80", fc.objects[1].GetName())
}
