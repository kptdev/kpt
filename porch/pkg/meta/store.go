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

package meta

import (
	"context"

	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	internalapi "github.com/GoogleContainerTools/kpt/porch/internal/api/porchinternal/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var tracer = otel.Tracer("meta")

const (
	PkgRevisionRepoLabel = "internal.porch.kpt.dev/repository"
)

// MetadataStore is the store for keeping metadata about PackageRevisions. Typical
// examples of metadata we want to keep is labels, annotations, owner references, and
// finalizers.
type MetadataStore interface {
	Get(ctx context.Context, namespacedName types.NamespacedName) (PackageRevisionMeta, error)
	List(ctx context.Context, repo *configapi.Repository) ([]PackageRevisionMeta, error)
	Create(ctx context.Context, pkgRevMeta PackageRevisionMeta, repo *configapi.Repository) (PackageRevisionMeta, error)
	Update(ctx context.Context, pkgRevMeta PackageRevisionMeta) (PackageRevisionMeta, error)
	Delete(ctx context.Context, namespacedName types.NamespacedName) (PackageRevisionMeta, error)
}

// PackageRevisionMeta contains metadata about a specific PackageRevision. The
// PackageRevision is identified by the name and namespace.
type PackageRevisionMeta struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

var _ MetadataStore = &crdMetadataStore{}

func NewCrdMetadataStore(coreClient client.Client) *crdMetadataStore {
	return &crdMetadataStore{
		coreClient: coreClient,
	}
}

// crdMetadataStore is an implementation of the MetadataStore interface that
// stores metadata in a CRD.
type crdMetadataStore struct {
	coreClient client.Client
}

func (c *crdMetadataStore) Get(ctx context.Context, namespacedName types.NamespacedName) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Get", trace.WithAttributes())
	defer span.End()

	var internalPkgRev internalapi.PackageRev
	err := c.coreClient.Get(ctx, namespacedName, &internalPkgRev)
	if err != nil {
		return PackageRevisionMeta{}, err
	}

	labels := internalPkgRev.Labels
	delete(labels, PkgRevisionRepoLabel)

	return PackageRevisionMeta{
		Name:        internalPkgRev.Name,
		Namespace:   internalPkgRev.Namespace,
		Labels:      labels,
		Annotations: internalPkgRev.Annotations,
	}, nil
}

func (c *crdMetadataStore) List(ctx context.Context, repo *configapi.Repository) ([]PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::List", trace.WithAttributes())
	defer span.End()

	var internalPkgRevList internalapi.PackageRevList
	err := c.coreClient.List(ctx, &internalPkgRevList, client.InNamespace(repo.Namespace), client.MatchingLabels(map[string]string{PkgRevisionRepoLabel: repo.Name}))
	if err != nil {
		return nil, err
	}
	var pkgRevMetas []PackageRevisionMeta
	var names []string
	for _, ipr := range internalPkgRevList.Items {
		labels := ipr.Labels
		delete(labels, PkgRevisionRepoLabel)
		pkgRevMetas = append(pkgRevMetas, PackageRevisionMeta{
			Name:        ipr.Name,
			Namespace:   ipr.Namespace,
			Labels:      labels,
			Annotations: ipr.Annotations,
		})
		names = append(names, ipr.Name)
	}
	return pkgRevMetas, nil
}

func (c *crdMetadataStore) Create(ctx context.Context, pkgRevMeta PackageRevisionMeta, repo *configapi.Repository) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Create", trace.WithAttributes())
	defer span.End()

	labels := pkgRevMeta.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[PkgRevisionRepoLabel] = repo.Name
	internalPkgRev := internalapi.PackageRev{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pkgRevMeta.Name,
			Namespace:   pkgRevMeta.Namespace,
			Labels:      labels,
			Annotations: pkgRevMeta.Annotations,
			// We probably should make these owner refs point to the PackageRevision CRs instead.
			// But we need to make sure that deletion of these are correctly picked up by the
			// GC. Currently we delete PackageRevisions through polling of the git/oci repos, and
			// that doesn't get picked up by the GC.
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: configapi.RepositoryGVK.GroupVersion().String(),
					Kind:       configapi.RepositoryGVK.Kind,
					Name:       repo.Name,
					UID:        repo.UID,
				},
			},
		},
	}
	if err := c.coreClient.Create(ctx, &internalPkgRev); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return c.Update(ctx, pkgRevMeta)
		}
		return PackageRevisionMeta{}, err
	}
	return PackageRevisionMeta{
		Name:        internalPkgRev.Name,
		Namespace:   internalPkgRev.Namespace,
		Labels:      internalPkgRev.Labels,
		Annotations: internalPkgRev.Annotations,
	}, nil
}

func (c *crdMetadataStore) Update(ctx context.Context, pkgRevMeta PackageRevisionMeta) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Update", trace.WithAttributes())
	defer span.End()

	var internalPkgRev internalapi.PackageRev
	namespacedName := types.NamespacedName{
		Name:      pkgRevMeta.Name,
		Namespace: pkgRevMeta.Namespace,
	}
	err := c.coreClient.Get(ctx, namespacedName, &internalPkgRev)
	if err != nil {
		return PackageRevisionMeta{}, err
	}

	var labels map[string]string
	if pkgRevMeta.Labels != nil {
		labels = pkgRevMeta.Labels
	} else {
		labels = make(map[string]string)
	}
	labels[PkgRevisionRepoLabel] = internalPkgRev.Labels[PkgRevisionRepoLabel]
	internalPkgRev.Labels = labels

	var annotations map[string]string
	if pkgRevMeta.Annotations != nil {
		annotations = pkgRevMeta.Annotations
	} else {
		annotations = make(map[string]string)
	}
	internalPkgRev.Annotations = annotations

	if err := c.coreClient.Update(ctx, &internalPkgRev); err != nil {
		return PackageRevisionMeta{}, err
	}
	delete(labels, PkgRevisionRepoLabel)
	return PackageRevisionMeta{
		Name:        pkgRevMeta.Name,
		Namespace:   pkgRevMeta.Namespace,
		Labels:      labels,
		Annotations: internalPkgRev.Annotations,
	}, nil
}

func (c *crdMetadataStore) Delete(ctx context.Context, namespacedName types.NamespacedName) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Delete", trace.WithAttributes())
	defer span.End()

	var internalPkgRev internalapi.PackageRev
	err := c.coreClient.Get(ctx, namespacedName, &internalPkgRev)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return PackageRevisionMeta{}, nil
		}
		return PackageRevisionMeta{}, err
	}

	if err := c.coreClient.Delete(ctx, &internalPkgRev); err != nil {
		if apierrors.IsNotFound(err) {
			return PackageRevisionMeta{}, nil
		}
		return PackageRevisionMeta{}, err
	}
	labels := internalPkgRev.Labels
	delete(labels, PkgRevisionRepoLabel)
	return PackageRevisionMeta{
		Name:        internalPkgRev.Name,
		Namespace:   internalPkgRev.Namespace,
		Labels:      labels,
		Annotations: internalPkgRev.Annotations,
	}, nil
}
