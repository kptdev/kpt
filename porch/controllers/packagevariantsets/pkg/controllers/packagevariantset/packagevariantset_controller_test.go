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

	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
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
		"missing upstream package, but has both revision and tag": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    revision: v1
    tag: main`,
			expectedErrs: []string{"spec.upstream.package is a required field",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream package repo, revision, and tag": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package:
      name: foo`,
			expectedErrs: []string{"spec.upstream.package.repo is a required field",
				"must have one of spec.upstream.revision and spec.upstream.tag",
				"must specify at least one item in spec.targets",
			},
		},
		"missing upstream package name, revision, and tag": {
			packageVariant: packageVariantHeader + `
spec:
  upstream:
    package:
      repo: foo`,
			expectedErrs: []string{"spec.upstream.package.name is a required field",
				"must have one of spec.upstream.revision and spec.upstream.tag",
				"must specify at least one item in spec.targets",
			},
		},
		"invalid targets": {
			packageVariant: packageVariantHeader + `
spec:
  targets:
  - package:
      name: foo
    packageName:
      name: 
        value: foo
  - package:
      repo: bar
    repositories:
      foo: bar
  - package:
      name: foo
      repo: bar
    objects:
      repoName:
        value: foo
      `,
			expectedErrs: []string{"spec.upstream is a required field",
				"spec.targets[0] cannot specify both fields `packageName` and `package`",
				"spec.targets[0].package.repo cannot be empty when using `package`",
				"spec.targets[1] must specify one of `package`, `repositories`, or `objects`",
				"spec.targets[2].objects must have at least one selector",
				"spec.targets[2] must specify one of `package`, `repositories`, or `objects`",
			},
		},
		"invalid adoption and deletion policies": {
			packageVariant: packageVariantHeader + `
spec:
  adoptionPolicy: invalid
  deletionPolicy: invalid
`,
			expectedErrs: []string{"spec.upstream is a required field",
				"must specify at least one item in spec.targets",
				"spec.adoptionPolicy: Invalid value: \"invalid\": field can only be \"adoptNone\" or \"adoptExisting\"",
				"spec.deletionPolicy: Invalid value: \"invalid\": field can only be \"orphan\" or \"delete\"",
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

func TestFetchValue(t *testing.T) {
	pod := kyaml.MustParse(`
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
  - name: foo
    image: image:1.2.3
    ports:
    - containerPort: 8080`)

	testCases := map[string]struct {
		input    *api.ValueOrFromField
		expected string
	}{
		"nil input": {
			input:    nil,
			expected: "",
		},
		"empty struct input": {
			input:    &api.ValueOrFromField{},
			expected: "",
		},
		"string literal value": {
			input: &api.ValueOrFromField{
				Value: "literal",
			},
			expected: "literal",
		},
		"value from field using key-value selector": {
			input: &api.ValueOrFromField{
				FromField: "spec.containers[name=foo].image",
			},
			expected: "image:1.2.3",
		},
		"value from field using integer selector": {
			input: &api.ValueOrFromField{
				FromField: "spec.containers[1].image",
			},
			expected: "image:1.2.3",
		},
	}

	for tn, tc := range testCases {
		r := &PackageVariantSetReconciler{}
		t.Run(tn, func(t *testing.T) {
			v, err := r.fetchValue(tc.input, pod)
			require.NoError(t, err)
			require.Equal(t, tc.expected, v)
		})
	}
}

func TestRepositorySet(t *testing.T) {
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

	var target api.Target
	require.NoError(t, yaml.Unmarshal([]byte(`
repositories:
  foo: bar
  abc: def
packageName:
  baseName:
    value: dpn`), &target))

	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true})
	r := PackageVariantSetReconciler{serializer: s}

	result, err := r.repositorySet(&target, "upn", &repoList)
	require.NoError(t, err)
	require.Equal(t, []*pkgvarapi.Downstream{{
		Repo:    "my-repo-1",
		Package: "dpn",
	}, {
		Repo:    "my-repo-2",
		Package: "dpn",
	},
	}, result)
}

func TestGetSelectedObjects(t *testing.T) {
	selectors := []api.Selector{{
		APIVersion: "v1",
		Kind:       "Pod",
		Labels:     &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
	}}
	reconciler := &PackageVariantSetReconciler{
		Client:     new(fakeClient),
		serializer: json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: true}),
	}
	selectedObjects, err := reconciler.getSelectedObjects(context.Background(), selectors)
	require.NoError(t, err)
	require.Equal(t, 1, len(selectedObjects))

	expectedResId := resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "Pod"), "my-pod-1", "")
	obj, found := selectedObjects[expectedResId]
	require.True(t, found)
	require.Equal(t, `apiVersion: v1
kind: Pod
metadata:
  labels:
    abc: def
    foo: bar
  name: my-pod-1
`, obj.MustString())
}

func TestObjectSet(t *testing.T) {
	selectedObjects := map[resid.ResId]*kyaml.RNode{
		resid.NewResIdWithNamespace(resid.NewGvk("", "v1", "Pod"), "my-pod-1", ""): kyaml.MustParse(`apiVersion: v1
kind: Pod
metadata:
  labels:
    repo: my-repo
  name: downstream
`),
	}

	target := &api.Target{
		PackageName: &api.PackageName{
			Name: &api.ValueOrFromField{FromField: "metadata.name"},
		},
		Objects: &api.ObjectSelector{
			RepoName: &api.ValueOrFromField{FromField: "metadata.labels.repo"},
		},
	}

	pvs := &PackageVariantSetReconciler{}
	objectSet, err := pvs.objectSet(target, "upstream", selectedObjects)
	require.NoError(t, err)
	require.Equal(t, len(objectSet), 1)
	require.Equal(t, pkgvarapi.Downstream{
		Repo:    "my-repo",
		Package: "downstream",
	}, *objectSet[0])
}

func TestEnsurePackageVariants(t *testing.T) {
	upstream := &pkgvarapi.Upstream{Repo: "up", Package: "up", Revision: "up"}
	downstreams := []*pkgvarapi.Downstream{
		{Repo: "dn-1", Package: "dn-1"},
		{Repo: "dn-3", Package: "dn-3"},
	}
	fc := &fakeClient{}
	reconciler := &PackageVariantSetReconciler{Client: fc}
	require.NoError(t, reconciler.ensurePackageVariants(context.Background(), upstream, downstreams,
		&api.PackageVariantSet{ObjectMeta: metav1.ObjectMeta{Name: "my-pvs"}}))
	require.Equal(t, 2, len(fc.objects))
	require.Equal(t, "my-pv-1", fc.objects[0].GetName())
	require.Equal(t, "my-pvs-28ace69e71f644931cd8cc1e8e9388f4de486901", fc.objects[1].GetName())
}
