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

func TestEnsurePackageVariants(t *testing.T) {
	downstreams := []pvContext{
		{repoDefault: "dn-1", packageDefault: "dn-1"},
		{repoDefault: "dn-3", packageDefault: "dn-3"},
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
	require.Equal(t, "my-pvs-9c07e0818b755a2067903d10aecf91e19dcb9a82", fc.objects[1].GetName())
}
