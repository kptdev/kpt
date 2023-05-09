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

package meta

import (
	"context"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	internalapi "github.com/GoogleContainerTools/kpt/porch/internal/api/porchinternal/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var tracer = otel.Tracer("meta")

const (
	PkgRevisionRepoLabel = "internal.porch.kpt.dev/repository"
	PkgRevisionFinalizer = "internal.porch.kpt.dev/packagerevision"
)

var (
	packageRevisionGVK = porchapi.SchemeGroupVersion.WithKind("PackageRevision")
)

// MetadataStore is the store for keeping metadata about PackageRevisions. Typical
// examples of metadata we want to keep is labels, annotations, owner references, and
// finalizers.
type MetadataStore interface {
	Get(ctx context.Context, namespacedName types.NamespacedName) (PackageRevisionMeta, error)
	List(ctx context.Context, repo *configapi.Repository) ([]PackageRevisionMeta, error)
	Create(ctx context.Context, pkgRevMeta PackageRevisionMeta, repoName string, pkgRevUID types.UID) (PackageRevisionMeta, error)
	Update(ctx context.Context, pkgRevMeta PackageRevisionMeta) (PackageRevisionMeta, error)
	Delete(ctx context.Context, namespacedName types.NamespacedName, clearFinalizer bool) (PackageRevisionMeta, error)
}

// PackageRevisionMeta contains metadata about a specific PackageRevision. The
// PackageRevision is identified by the name and namespace.
type PackageRevisionMeta struct {
	Name              string
	Namespace         string
	Labels            map[string]string
	Annotations       map[string]string
	Finalizers        []string
	OwnerReferences   []metav1.OwnerReference
	DeletionTimestamp *metav1.Time
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

	return toPackageRevisionMeta(&internalPkgRev), nil
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
	for _, ipr := range internalPkgRevList.Items {
		pkgRevMetas = append(pkgRevMetas, toPackageRevisionMeta(&ipr))
	}
	return pkgRevMetas, nil
}

func (c *crdMetadataStore) Create(ctx context.Context, pkgRevMeta PackageRevisionMeta, repoName string, pkgRevUID types.UID) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Create", trace.WithAttributes())
	defer span.End()

	labels := pkgRevMeta.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[PkgRevisionRepoLabel] = repoName

	ownerReferences := append(pkgRevMeta.OwnerReferences, metav1.OwnerReference{
		APIVersion: packageRevisionGVK.GroupVersion().String(),
		Kind:       packageRevisionGVK.Kind,
		Name:       pkgRevMeta.Name,
		UID:        pkgRevUID,
	})

	finalizers := append(pkgRevMeta.Finalizers, PkgRevisionFinalizer)

	internalPkgRev := internalapi.PackageRev{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pkgRevMeta.Name,
			Namespace:       pkgRevMeta.Namespace,
			Labels:          labels,
			Annotations:     pkgRevMeta.Annotations,
			Finalizers:      finalizers,
			OwnerReferences: ownerReferences,
		},
	}
	klog.Infof("Creating packagerev %s/%s", internalPkgRev.Namespace, internalPkgRev.Name)
	if err := c.coreClient.Create(ctx, &internalPkgRev); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return c.Update(ctx, pkgRevMeta)
		}
		return PackageRevisionMeta{}, err
	}
	return toPackageRevisionMeta(&internalPkgRev), nil
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

	// Copy updated labels to the CR and add the repository label
	// that is only used on the CR.
	var labels map[string]string
	if pkgRevMeta.Labels != nil {
		labels = pkgRevMeta.Labels
	} else {
		labels = make(map[string]string)
	}
	labels[PkgRevisionRepoLabel] = internalPkgRev.Labels[PkgRevisionRepoLabel]
	internalPkgRev.Labels = labels
	internalPkgRev.Annotations = pkgRevMeta.Annotations

	// Copy update ownerReferences to the CR and make sure to also
	// add the ownerReferences pointing to the PackageRevision.
	ownerReferences := pkgRevMeta.OwnerReferences
	for _, or := range internalPkgRev.OwnerReferences {
		if isPackageRevOwnerRef(or, internalPkgRev.Name) {
			ownerReferences = append(ownerReferences, or)
		}
	}
	internalPkgRev.OwnerReferences = ownerReferences
	internalPkgRev.Finalizers = append(pkgRevMeta.Finalizers, PkgRevisionFinalizer)

	klog.Infof("Updating packagerev %s/%s", internalPkgRev.Namespace, internalPkgRev.Name)
	if err := c.coreClient.Update(ctx, &internalPkgRev); err != nil {
		return PackageRevisionMeta{}, err
	}
	return toPackageRevisionMeta(&internalPkgRev), nil
}

func (c *crdMetadataStore) Delete(ctx context.Context, namespacedName types.NamespacedName, clearFinalizers bool) (PackageRevisionMeta, error) {
	ctx, span := tracer.Start(ctx, "crdMetadataStore::Delete", trace.WithAttributes())
	defer span.End()

	var internalPkgRev internalapi.PackageRev
	err := c.coreClient.Get(ctx, namespacedName, &internalPkgRev)
	if err != nil {
		return PackageRevisionMeta{}, err
	}

	if clearFinalizers {
		internalPkgRev.Finalizers = []string{}
		if err = c.coreClient.Update(ctx, &internalPkgRev); err != nil {
			return PackageRevisionMeta{}, err
		}
	}

	klog.Infof("Deleting packagerev %s/%s", internalPkgRev.Namespace, internalPkgRev.Name)
	if err := c.coreClient.Delete(ctx, &internalPkgRev); err != nil {
		return PackageRevisionMeta{}, err
	}
	return toPackageRevisionMeta(&internalPkgRev), nil
}

func toPackageRevisionMeta(internalPkgRev *internalapi.PackageRev) PackageRevisionMeta {
	labels := internalPkgRev.Labels
	delete(labels, PkgRevisionRepoLabel)

	var ownerReferences []metav1.OwnerReference
	for _, or := range internalPkgRev.OwnerReferences {
		// Don't include ownerReference to the PackageRevision itself. It is
		// only used by Porch internally.
		if !isPackageRevOwnerRef(or, internalPkgRev.Name) {
			ownerReferences = append(ownerReferences, or)
		}
	}

	var finalizers []string
	for _, f := range internalPkgRev.Finalizers {
		if f != PkgRevisionFinalizer {
			finalizers = append(finalizers, f)
		}
	}

	return PackageRevisionMeta{
		Name:              internalPkgRev.Name,
		Namespace:         internalPkgRev.Namespace,
		Labels:            labels,
		Annotations:       internalPkgRev.Annotations,
		Finalizers:        finalizers,
		OwnerReferences:   ownerReferences,
		DeletionTimestamp: internalPkgRev.DeletionTimestamp,
	}
}

func isPackageRevOwnerRef(or metav1.OwnerReference, pkgRevName string) bool {
	return or.APIVersion == packageRevisionGVK.GroupVersion().String() &&
		or.Kind == packageRevisionGVK.Kind &&
		or.Name == pkgRevName
}
