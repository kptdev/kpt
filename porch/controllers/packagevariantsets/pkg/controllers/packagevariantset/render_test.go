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

var repoListYaml = []byte(`
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
`)

func TestRenderPackageVariantSpec(t *testing.T) {
	var repoList configapi.RepositoryList
	require.NoError(t, yaml.Unmarshal(repoListYaml, &repoList))

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
				repoDefault:    "my-repo-1",
				packageDefault: "p",
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "my-repo-1",
					Package: "p",
				},
			},
			expectedErrs: nil,
		},
		"template downstream.repo": {
			downstream: pvContext{
				repoDefault:    "my-repo-1",
				packageDefault: "p",
				template: &api.PackageVariantTemplate{
					Downstream: &api.DownstreamTemplate{
						Repo: pointer.String("my-repo-2"),
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "my-repo-2",
					Package: "p",
				},
			},
			expectedErrs: nil,
		},
		"template downstream.package": {
			downstream: pvContext{
				repoDefault:    "my-repo-1",
				packageDefault: "p",
				template: &api.PackageVariantTemplate{
					Downstream: &api.DownstreamTemplate{
						Package: pointer.String("new-p"),
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "my-repo-1",
					Package: "new-p",
				},
			},
			expectedErrs: nil,
		},
		"template adoption and deletion": {
			downstream: pvContext{
				repoDefault:    "my-repo-1",
				packageDefault: "p",
				template: &api.PackageVariantTemplate{
					AdoptionPolicy: &adoptExisting,
					DeletionPolicy: &deletionPolicyDelete,
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "my-repo-1",
					Package: "p",
				},
				AdoptionPolicy: "adoptExisting",
				DeletionPolicy: "delete",
			},
			expectedErrs: nil,
		},
		"template static labels and annotations": {
			downstream: pvContext{
				repoDefault:    "my-repo-1",
				packageDefault: "p",
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
					Repo:    "my-repo-1",
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
				repoDefault:    "my-repo-1",
				packageDefault: "p",
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
					Repo:    "my-repo-1",
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
		"template downstream with expressions": {
			downstream: pvContext{
				repoDefault:    "my-repo-1",
				packageDefault: "p",
				template: &api.PackageVariantTemplate{
					Downstream: &api.DownstreamTemplate{
						RepoExpr:    pointer.String("'my-repo-2'"),
						PackageExpr: pointer.String("repoDefault + '-' + packageDefault"),
					},
				},
			},
			expectedSpec: pkgvarapi.PackageVariantSpec{
				Upstream: pvs.Spec.Upstream,
				Downstream: &pkgvarapi.Downstream{
					Repo:    "my-repo-2",
					Package: "my-repo-1-p",
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

func TestEvalExpr(t *testing.T) {
	baseInputs := map[string]interface{}{
		"repoDefault":    "foo-repo",
		"packageDefault": "bar-package",
	}
	var repoList configapi.RepositoryList
	require.NoError(t, yaml.Unmarshal(repoListYaml, &repoList))

	r1Input, err := objectToInput(&repoList.Items[0])
	require.NoError(t, err)

	testCases := map[string]struct {
		expr           string
		target         interface{}
		expectedResult string
		expectedErr    string
	}{
		"no vars": {
			expr:           "'foo'",
			expectedResult: "foo",
			expectedErr:    "",
		},
		"repoDefault": {
			expr:           "repoDefault",
			expectedResult: "foo-repo",
			expectedErr:    "",
		},
		"packageDefault": {
			expr:           "packageDefault",
			expectedResult: "bar-package",
			expectedErr:    "",
		},
		"concat defaults": {
			expr:           "packageDefault + '-' + repoDefault",
			expectedResult: "bar-package-foo-repo",
			expectedErr:    "",
		},
		"repositories target": {
			expr: "target.repo + '/' + target.package",
			target: map[string]any{
				"repo":    "my-repo",
				"package": "my-package",
			},
			expectedResult: "my-repo/my-package",
			expectedErr:    "",
		},
		"repository target": {
			expr:           "target.name + '/' + target.labels['foo']",
			target:         r1Input,
			expectedResult: "my-repo-1/bar",
			expectedErr:    "",
		},
		"bad variable": {
			expr:        "badvar",
			expectedErr: "ERROR: <input>:1:1: undeclared reference to 'badvar' (in container '')\n | badvar\n | ^",
		},
		"bad expr": {
			expr:        "/",
			expectedErr: "ERROR: <input>:1:1: Syntax error: mismatched input '/' expecting {'[', '{', '(', '.', '-', '!', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}\n | /\n | ^\nERROR: <input>:1:2: Syntax error: mismatched input '<EOF>' expecting {'[', '{', '(', '.', '-', '!', 'true', 'false', 'null', NUM_FLOAT, NUM_INT, NUM_UINT, STRING, BYTES, IDENTIFIER}\n | /\n | .^",
		},
		"missing label": {
			expr:        "target.name + '/' + target.labels['no-such-label']",
			target:      r1Input,
			expectedErr: "no such key: no-such-label",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			inputs := map[string]any{}
			for k, v := range baseInputs {
				inputs[k] = v
			}
			inputs["target"] = tc.target
			val, err := evalExpr(tc.expr, inputs)
			if tc.expectedErr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, val)
			} else {
				require.EqualError(t, err, tc.expectedErr)
			}
		})
	}
}

func TestOverlayMapExpr(t *testing.T) {
	baseInputs := map[string]interface{}{
		"repoDefault":    "foo-repo",
		"packageDefault": "bar-package",
	}

	testCases := map[string]struct {
		inMap          map[string]string
		mapExprs       []api.MapExpr
		expectedResult map[string]string
		expectedErr    string
	}{
		"empty starting map": {
			inMap: map[string]string{},
			mapExprs: []api.MapExpr{
				{
					Key:   pointer.String("foo"),
					Value: pointer.String("bar"),
				},
				{
					KeyExpr: pointer.String("repoDefault"),
					Value:   pointer.String("barbar"),
				},
				{
					Key:       pointer.String("bar"),
					ValueExpr: pointer.String("packageDefault"),
				},
			},
			expectedResult: map[string]string{
				"foo":      "bar",
				"foo-repo": "barbar",
				"bar":      "bar-package",
			},
		},
		"static overlay": {
			inMap: map[string]string{
				"foo": "bar",
				"bar": "foo",
			},
			mapExprs: []api.MapExpr{
				{
					Key:   pointer.String("foo"),
					Value: pointer.String("new-bar"),
				},
				{
					Key:   pointer.String("foofoo"),
					Value: pointer.String("barbar"),
				},
			},
			expectedResult: map[string]string{
				"foo":    "new-bar",
				"bar":    "foo",
				"foofoo": "barbar",
			},
		},
		"exprs overlay": {
			inMap: map[string]string{
				"foo": "bar",
				"bar": "foo",
			},
			mapExprs: []api.MapExpr{
				{
					KeyExpr: pointer.String("'foo'"),
					Value:   pointer.String("new-bar"),
				},
				{
					Key:       pointer.String("bar"),
					ValueExpr: pointer.String("packageDefault"),
				},
			},
			expectedResult: map[string]string{
				"foo": "new-bar",
				"bar": "bar-package",
			},
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			err := overlayMapExpr(tc.inMap, tc.mapExprs, baseInputs)
			if tc.expectedErr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.expectedResult, tc.inMap)
			} else {
				require.EqualError(t, err, tc.expectedErr)
			}
		})
	}
}
