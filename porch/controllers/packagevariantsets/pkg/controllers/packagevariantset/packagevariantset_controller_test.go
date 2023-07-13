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
	"sort"
	"testing"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	pkgvarapi "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariants/api/v1alpha1"
	api "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/yaml"
)

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

func TestUnrollDownstreamTargets(t *testing.T) {
	pvs := &api.PackageVariantSet{
		ObjectMeta: metav1.ObjectMeta{Name: "my-pvs"},
		Spec: api.PackageVariantSetSpec{
			Upstream: &pkgvarapi.Upstream{Repo: "up", Package: "up", Revision: "up"},
			Targets: []api.Target{
				{
					Repositories: []api.RepositoryTarget{
						{Name: "r1", PackageNames: []string{"p1", "p2", "p3"}},
						{Name: "r2"},
					},
				},
				{
					RepositorySelector: &metav1.LabelSelector{},
				},
			},
		},
	}

	fc := &fakeClient{}
	reconciler := &PackageVariantSetReconciler{Client: fc}
	downstreams, err := reconciler.unrollDownstreamTargets(context.Background(), pvs)
	require.NoError(t, err)
	require.Equal(t, 6, len(downstreams))
	require.Equal(t, downstreams[0].repoDefault, "r1")
	require.Equal(t, downstreams[0].packageDefault, "p1")
	require.Equal(t, downstreams[1].repoDefault, "r1")
	require.Equal(t, downstreams[1].packageDefault, "p2")
	require.Equal(t, downstreams[2].repoDefault, "r1")
	require.Equal(t, downstreams[2].packageDefault, "p3")

	require.Equal(t, downstreams[3].repoDefault, "r2")
	require.Equal(t, downstreams[3].packageDefault, "up")

	// from the RepositorySelector, but fake client returns pods anyway
	require.Equal(t, downstreams[4].repoDefault, "my-pod-1")
	require.Equal(t, downstreams[4].packageDefault, "up")
	require.Equal(t, downstreams[4].object.GetName(), "my-pod-1")

	require.Equal(t, downstreams[5].repoDefault, "my-pod-2")
	require.Equal(t, downstreams[5].packageDefault, "up")
	require.Equal(t, downstreams[5].object.GetName(), "my-pod-2")

}

func TestEnsurePackageVariants(t *testing.T) {
	downstreams := []pvContext{
		{repoDefault: "dnrepo2", packageDefault: "dnpkg2"},
		{repoDefault: "dnrepo3", packageDefault: "dnpkg3"},
		{repoDefault: "dnrepo4", packageDefault: "supersupersuperloooooooooooooooooooooooooooooooooooooooooooooooooongpkgname"},
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
	require.Equal(t, 1, len(fc.deleted))
	require.Equal(t, "my-pvs-dnrepo1-dnpkg1", fc.deleted[0].GetName())
	require.Equal(t, 1, len(fc.updated))
	require.Equal(t, "my-pvs-dnrepo2-dnpkg2", fc.updated[0].GetName())
	require.Equal(t, 2, len(fc.created))
	// ordering of calls to create is not stable (map iteration)
	sort.Slice(fc.created, func(i, j int) bool {
		return fc.created[i].GetName() < fc.created[j].GetName()
	})
	require.Equal(t, "my-pvs-dnrepo3-dnpkg3", fc.created[0].GetName())
	require.Equal(t, "my-pvs-dnrepo4-supersupersuperlooooooooooooooooooooooo-bec36506", fc.created[1].GetName())
}
