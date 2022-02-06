// Copyright 2022 Google LLC
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

package e2e

import (
	"context"
	"reflect"
	"strings"
	"testing"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	coreapi "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func TestE2E(t *testing.T) {
	Run(&PorchSuite{}, t)
}

func Run(suite interface{}, t *testing.T) {
	sv := reflect.ValueOf(suite)
	st := reflect.TypeOf(suite)
	ctx := context.Background()

	t.Run(st.Elem().Name(), func(t *testing.T) {
		if init, ok := suite.(Initializer); ok {
			init.Initialize(ctx, t)
		}

		var ts *TestSuite = sv.Elem().FieldByName("TestSuite").Addr().Interface().(*TestSuite)

		for i, max := 0, st.NumMethod(); i < max; i++ {
			m := st.Method(i)
			if strings.HasPrefix(m.Name, "Test") {
				t.Run(m.Name, func(t *testing.T) {
					ts.T = t
					m.Func.Call([]reflect.Value{sv, reflect.ValueOf(ctx)})
				})
			}
		}
	})
}

type PorchSuite struct {
	TestSuite
}

var _ Initializer = &PorchSuite{}

func (t *PorchSuite) TestGitRepository(ctx context.Context) {
	if t.ptc.Git.Repo == "" {
		t.Skipf("Skipping TestGitRepository; no Git repository specified.")
	}

	var secret string
	if t.ptc.Git.Username != "" || t.ptc.Git.Token != "" {
		const credSecret = "git-repository-auth"
		immutable := true
		t.CreateF(ctx, &coreapi.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      credSecret,
				Namespace: t.namespace,
			},
			Immutable: &immutable,
			Data: map[string][]byte{
				"username": []byte(t.ptc.Git.Username),
				"token":    []byte(t.ptc.Git.Token),
			},
			// TODO: Store as SecretTypeBasicAuth ?
			Type: coreapi.SecretTypeOpaque,
		})

		secret = credSecret

		t.Cleanup(func() {
			t.DeleteL(ctx, &coreapi.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      credSecret,
					Namespace: t.namespace,
				},
			})
		})
	}

	t.CreateF(ctx, &configapi.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "git",
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Title:       "Porch Test Repository",
			Description: "Porch Test Repository Description",
			Type:        configapi.RepositoryTypeGit,
			Content:     "PackageRevision",
			Git: &configapi.GitRepository{
				Repo:   t.ptc.Git.Repo,
				Branch: "main",
				SecretRef: configapi.SecretRef{
					Name: secret,
				},
			},
		},
	})

	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "git",
				Namespace: t.namespace,
			},
		})
	})
}

func (t *PorchSuite) TestFunctionRepository(ctx context.Context) {
	t.CreateF(ctx, &configapi.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "function-repository",
			Namespace: t.namespace,
		},
		Spec: configapi.RepositorySpec{
			Title:       "Function Repository",
			Description: "Test Function Repository",
			Type:        configapi.RepositoryTypeOCI,
			Content:     "Function",
			Oci: &configapi.OciRepository{
				Registry: "gcr.io/kpt-fn",
			},
		},
	})

	t.Cleanup(func() {
		t.DeleteL(ctx, &configapi.Repository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "function-repository",
				Namespace: t.namespace,
			},
		})
	})

	list := &porchapi.FunctionList{}
	t.ListE(ctx, list)

	if got := len(list.Items); got == 0 {
		t.Errorf("Found no functions in gcr.io/kpt-fn repository; expected at least one")
	}
}
