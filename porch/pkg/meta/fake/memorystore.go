// Copyright 2022 The kpt Authors
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

package fake

import (
	"context"

	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// MemoryMetadataStore is an in-memory implementation of the MetadataStore interface. It
// means metadata about packagerevisions will be stored in memory, which is useful for testing.
type MemoryMetadataStore struct {
	Metas []meta.PackageRevisionMeta
}

var _ meta.MetadataStore = &MemoryMetadataStore{}

func (m *MemoryMetadataStore) Get(ctx context.Context, namespacedName types.NamespacedName) (meta.PackageRevisionMeta, error) {
	for _, meta := range m.Metas {
		if meta.Name == namespacedName.Name && meta.Namespace == namespacedName.Namespace {
			return meta, nil
		}
	}
	return meta.PackageRevisionMeta{}, apierrors.NewNotFound(
		schema.GroupResource{Group: "config.kpt.dev", Resource: "packagerevisions"},
		namespacedName.Name,
	)
}

func (m *MemoryMetadataStore) List(ctx context.Context, repo *configapi.Repository) ([]meta.PackageRevisionMeta, error) {
	return m.Metas, nil
}

func (m *MemoryMetadataStore) Create(ctx context.Context, pkgRevMeta meta.PackageRevisionMeta, repoName string, pkgRevUID types.UID) (meta.PackageRevisionMeta, error) {
	for _, m := range m.Metas {
		if m.Name == pkgRevMeta.Name && m.Namespace == pkgRevMeta.Namespace {
			return m, apierrors.NewAlreadyExists(
				schema.GroupResource{Group: "config.kpt.dev", Resource: "packagerevisions"},
				m.Name,
			)
		}
	}
	m.Metas = append(m.Metas, pkgRevMeta)
	return pkgRevMeta, nil
}

func (m *MemoryMetadataStore) Update(ctx context.Context, pkgRevMeta meta.PackageRevisionMeta) (meta.PackageRevisionMeta, error) {
	i := -1
	for j, m := range m.Metas {
		if m.Name == pkgRevMeta.Name && m.Namespace == pkgRevMeta.Namespace {
			i = j
		}
	}
	if i < 0 {
		return meta.PackageRevisionMeta{}, apierrors.NewNotFound(
			schema.GroupResource{Group: "config.porch.kpt.dev", Resource: "packagerevisions"},
			pkgRevMeta.Name,
		)
	}
	m.Metas[i] = pkgRevMeta
	return pkgRevMeta, nil
}

func (m *MemoryMetadataStore) Delete(ctx context.Context, namespacedName types.NamespacedName, _ bool) (meta.PackageRevisionMeta, error) {
	var metas []meta.PackageRevisionMeta
	found := false
	var deletedMeta meta.PackageRevisionMeta
	for _, m := range m.Metas {
		if m.Name == namespacedName.Name && m.Namespace == namespacedName.Namespace {
			found = true
			deletedMeta = m
		} else {
			metas = append(metas, m)
		}
	}
	if !found {
		return meta.PackageRevisionMeta{}, apierrors.NewNotFound(
			schema.GroupResource{Group: "config.kpt.dev", Resource: "packagerevisions"},
			namespacedName.Name,
		)
	}
	m.Metas = metas
	return deletedMeta, nil
}
