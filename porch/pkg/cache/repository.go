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

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
)

var tracer = otel.Tracer("cache")

// We take advantage of the cache having a global view of all the packages
// in a repository and compute the latest package revision in the cache
// rather than add another level of caching in the repositories themselves.
// This also reuses the revision comparison code and ensures same behavior
// between Git and OCI.

var _ repository.Repository = &cachedRepository{}
var _ repository.FunctionRepository = &cachedRepository{}

type cachedRepository struct {
	id string
	// We need the kubernetes object so we can add the appropritate
	// ownerreferences to PackageRevision resources.
	repoSpec *configapi.Repository
	repo     repository.Repository
	cancel   context.CancelFunc

	mutex                  sync.Mutex
	cachedPackageRevisions map[repository.PackageRevisionKey]*cachedPackageRevision
	cachedPackages         map[repository.PackageKey]*cachedPackage

	// TODO: Currently we support repositories with homogenous content (only packages xor functions). Model this more optimally?
	cachedFunctions []repository.Function
	// Error encountered on repository refresh by the refresh goroutine.
	// This is returned back by the cache to the background goroutine when it calls periodicall to resync repositories.
	refreshRevisionsError error
	refreshPkgsError      error

	objectNotifier objectNotifier

	metadataStore meta.MetadataStore
}

func newRepository(id string, repoSpec *configapi.Repository, repo repository.Repository, objectNotifier objectNotifier, metadataStore meta.MetadataStore) *cachedRepository {
	ctx, cancel := context.WithCancel(context.Background())
	r := &cachedRepository{
		id:             id,
		repoSpec:       repoSpec,
		repo:           repo,
		cancel:         cancel,
		objectNotifier: objectNotifier,
		metadataStore:  metadataStore,
	}

	// TODO: Should we fetch the packages here?

	go r.pollForever(ctx)

	return r
}

func (r *cachedRepository) ListPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	packages, err := r.getPackageRevisions(ctx, filter, false)
	if err != nil {
		return nil, err
	}

	return packages, nil
}

func (r *cachedRepository) ListFunctions(ctx context.Context) ([]repository.Function, error) {
	functions, err := r.getFunctions(ctx, false)
	if err != nil {
		return nil, err
	}
	return functions, nil
}

func (r *cachedRepository) getRefreshError() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// TODO: This should also check r.refreshPkgsError when
	//   the package resource is fully supported.

	return r.refreshRevisionsError
}

func (r *cachedRepository) getPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter, forceRefresh bool) ([]repository.PackageRevision, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, packageRevisions, err := r.getCachedPackages(ctx, forceRefresh)
	if err != nil {
		return nil, err
	}

	return toPackageRevisionSlice(packageRevisions, filter), nil
}

func (r *cachedRepository) getPackages(ctx context.Context, filter repository.ListPackageFilter, forceRefresh bool) ([]repository.Package, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	packages, _, err := r.getCachedPackages(ctx, forceRefresh)
	if err != nil {
		return nil, err
	}

	return toPackageSlice(packages, filter), nil
}

// getCachedPackages returns cachedPackages; fetching it if not cached or if forceRefresh.
// mutex must be held.
func (r *cachedRepository) getCachedPackages(ctx context.Context, forceRefresh bool) (map[repository.PackageKey]*cachedPackage, map[repository.PackageRevisionKey]*cachedPackageRevision, error) {
	// must hold mutex
	packages := r.cachedPackages
	packageRevisions := r.cachedPackageRevisions
	err := r.refreshRevisionsError

	if forceRefresh {
		packages = nil
		packageRevisions = nil
	}

	if packages == nil {
		packages, packageRevisions, err = r.refreshAllCachedPackages(ctx)
	}

	return packages, packageRevisions, err
}

func (r *cachedRepository) getFunctions(ctx context.Context, force bool) ([]repository.Function, error) {
	var functions []repository.Function

	if !force {
		r.mutex.Lock()
		functions = r.cachedFunctions
		r.mutex.Unlock()
	}

	if functions == nil {
		fr, ok := (r.repo).(repository.FunctionRepository)
		if !ok {
			return []repository.Function{}, nil
		}

		if f, err := fr.ListFunctions(ctx); err != nil {
			return nil, err
		} else {
			functions = f
		}

		r.mutex.Lock()
		r.cachedFunctions = functions
		r.mutex.Unlock()
	}

	return functions, nil
}

func (r *cachedRepository) CreatePackageRevision(ctx context.Context, obj *v1alpha1.PackageRevision) (repository.PackageDraft, error) {
	created, err := r.repo.CreatePackageRevision(ctx, obj)
	if err != nil {
		return nil, err
	}

	return &cachedDraft{
		PackageDraft: created,
		cache:        r,
	}, nil
}

func (r *cachedRepository) UpdatePackageRevision(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	// Unwrap
	unwrapped := old.(*cachedPackageRevision).PackageRevision
	created, err := r.repo.UpdatePackageRevision(ctx, unwrapped)
	if err != nil {
		return nil, err
	}

	return &cachedDraft{
		PackageDraft: created,
		cache:        r,
	}, nil
}

func (r *cachedRepository) update(ctx context.Context, updated repository.PackageRevision) (*cachedPackageRevision, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// TODO: Technically we only need this package, not all packages
	if _, _, err := r.getCachedPackages(ctx, false); err != nil {
		klog.Warningf("failed to get cached packages: %v", err)
		// TODO: Invalidate all watches? We're dropping an add/update event
		return nil, err
	}

	k := updated.Key()
	// previous := r.cachedPackageRevisions[k]

	if updated.Lifecycle() == v1alpha1.PackageRevisionLifecyclePublished {
		oldKey := repository.PackageRevisionKey{
			Repository:    k.Repository,
			Package:       k.Package,
			WorkspaceName: k.WorkspaceName,
		}
		if _, ok := r.cachedPackageRevisions[oldKey]; ok {
			delete(r.cachedPackageRevisions, oldKey)
		}
	}

	cached := &cachedPackageRevision{PackageRevision: updated}
	r.cachedPackageRevisions[k] = cached

	// Recompute latest package revisions.
	// TODO: Just updated package?
	identifyLatestRevisions(r.cachedPackageRevisions)

	// TODO: Update the latest revisions for the r.cachedPackages
	return cached, nil
}

func (r *cachedRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	// Unwrap
	unwrapped := old.(*cachedPackageRevision).PackageRevision
	if err := r.repo.DeletePackageRevision(ctx, unwrapped); err != nil {
		return err
	}

	r.mutex.Lock()
	if r.cachedPackages != nil {
		k := old.Key()
		// previous := r.cachedPackages[k]
		delete(r.cachedPackageRevisions, k)

		// Recompute latest package revisions.
		// TODO: Only for affected object / key?
		identifyLatestRevisions(r.cachedPackageRevisions)
	}

	r.mutex.Unlock()

	return nil
}

func (r *cachedRepository) ListPackages(ctx context.Context, filter repository.ListPackageFilter) ([]repository.Package, error) {
	packages, err := r.getPackages(ctx, filter, false)
	if err != nil {
		return nil, err
	}

	return packages, nil
}

func (r *cachedRepository) CreatePackage(ctx context.Context, obj *v1alpha1.Package) (repository.Package, error) {
	klog.Infoln("cachedRepository::CreatePackage")
	return r.repo.CreatePackage(ctx, obj)
}

func (r *cachedRepository) DeletePackage(ctx context.Context, old repository.Package) error {
	// Unwrap
	unwrapped := old.(*cachedPackage).Package
	if err := r.repo.DeletePackage(ctx, unwrapped); err != nil {
		return err
	}

	// TODO: Do something more efficient than a full cache flush
	r.flush()

	return nil
}

func (r *cachedRepository) Close() error {
	r.cancel()
	return nil
}

// pollForever will continue polling until signal channel is closed or ctx is done.
func (r *cachedRepository) pollForever(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ticker.C:
			r.pollOnce(ctx)

		case <-ctx.Done():
			klog.V(2).Infof("exiting repository poller, because context is done: %v", ctx.Err())
			return
		}
	}
}

func (r *cachedRepository) pollOnce(ctx context.Context) {
	klog.Infof("background-refreshing repo %q", r.id)
	ctx, span := tracer.Start(ctx, "Repository::pollOnce", trace.WithAttributes())
	defer span.End()

	if _, err := r.getPackageRevisions(ctx, repository.ListPackageRevisionFilter{}, true); err != nil {
		klog.Warningf("error polling repo packages %s: %v", r.id, err)
	}
	// TODO: Uncomment when package resources are fully supported
	//if _, err := r.getPackages(ctx, repository.ListPackageRevisionFilter{}, true); err != nil {
	//	klog.Warningf("error polling repo packages %s: %v", r.id, err)
	//}
	if _, err := r.getFunctions(ctx, true); err != nil {
		klog.Warningf("error polling repo functions %s: %v", r.id, err)
	}
}

func (r *cachedRepository) flush() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cachedPackageRevisions = nil
	r.cachedPackages = nil
}

// refreshAllCachedPackages updates the cached map for this repository with all the newPackages,
// it also triggers notifications for all package changes.
// mutex must be held.
func (r *cachedRepository) refreshAllCachedPackages(ctx context.Context) (map[repository.PackageKey]*cachedPackage, map[repository.PackageRevisionKey]*cachedPackageRevision, error) {
	// TODO: Avoid simultaneous fetches?
	// TODO: Push-down partial refresh?

	// Look up all existing PackageRevCRs so we an compare those to the
	// actual Packagerevisions found in git/oci, and add/prune PackageRevCRs
	// as necessary.
	existingPkgRevCRs, err := r.metadataStore.List(ctx, r.repoSpec)
	if err != nil {
		return nil, nil, err
	}
	// Create a map so we can quickly check if a specific PackageRevisionMeta exists.
	existingPkgRevCRsMap := make(map[string]meta.PackageRevisionMeta)
	for i := range existingPkgRevCRs {
		pr := existingPkgRevCRs[i]
		existingPkgRevCRsMap[pr.Name] = pr
	}

	// TODO: Can we avoid holding the lock for the ListPackageRevisions / identifyLatestRevisions section?
	newPackageRevisions, err := r.repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		return nil, nil, fmt.Errorf("error listing packages: %w", err)
	}

	newPackageRevisionMap := make(map[repository.PackageRevisionKey]*cachedPackageRevision, len(newPackageRevisions))
	newPackageRevisionNames := make(map[string]bool)
	for _, newPackage := range newPackageRevisions {
		k := newPackage.Key()
		if newPackageRevisionMap[k] != nil {
			klog.Warningf("found duplicate packages with key %v", k)
		}

		newPackageRevisionMap[k] = &cachedPackageRevision{
			PackageRevision:  newPackage,
			isLatestRevision: false,
		}
		newPackageRevisionNames[newPackage.KubeObjectName()] = true
	}

	identifyLatestRevisions(newPackageRevisionMap)

	newPackageMap := make(map[repository.PackageKey]*cachedPackage)

	for _, newPackageRevision := range newPackageRevisionMap {
		if !newPackageRevision.isLatestRevision {
			continue
		}
		// TODO: Build package?
		// newPackage := &cachedPackage{
		// }
		// newPackageMap[newPackage.Key()] = newPackage
	}

	oldPackageRevisions := r.cachedPackageRevisions
	r.cachedPackageRevisions = newPackageRevisionMap
	r.cachedPackages = newPackageMap

	// We go through all PackageRev CRs that represents PackageRevisions
	// in the current repo and make sure they all have a corresponding
	// PackageRevision. The ones that doesn't is removed.
	for _, prm := range existingPkgRevCRs {
		if _, found := newPackageRevisionNames[prm.Name]; !found {
			if _, err := r.metadataStore.Delete(ctx, types.NamespacedName{
				Name:      prm.Name,
				Namespace: prm.Namespace,
			}); err != nil {
				if !apierrors.IsNotFound(err) {
					// This will be retried the next time the sync runs.
					klog.Warningf("unable to create PackageRev CR for %s/%s: %w",
						prm.Name, prm.Namespace, err)
				}
			}
		}
	}

	// We go through all the PackageRevisions and make sure they have
	// a corresponding PackageRev CR.
	for pkgRevName := range newPackageRevisionNames {
		if _, found := existingPkgRevCRsMap[pkgRevName]; !found {
			pkgRevMeta := meta.PackageRevisionMeta{
				Name:      pkgRevName,
				Namespace: r.repoSpec.Namespace,
			}
			if _, err := r.metadataStore.Create(ctx, pkgRevMeta, r.repoSpec); err != nil {
				// TODO: We should try to find a way to make these errors available through
				// either the repository CR or the PackageRevision CR. This will be
				// retried on the next sync.
				klog.Warningf("unable to create PackageRev CR for %s/%s: %w",
					r.repoSpec.Namespace, pkgRevName, err)
			}
		}
	}

	// Send notification for packages that changed.
	for k, newPackage := range r.cachedPackageRevisions {
		oldPackage := oldPackageRevisions[k]
		metaPackage, found := existingPkgRevCRsMap[newPackage.KubeObjectName()]
		if !found {
			klog.Warningf("no PackageRev CR found for PackageRevision %s", newPackage.KubeObjectName())
		}
		if oldPackage == nil {
			r.objectNotifier.NotifyPackageRevisionChange(watch.Added, newPackage, metaPackage)
		} else {
			// TODO: only if changed
			klog.Warningf("over-notifying of package updates (even on unchanged packages)")
			r.objectNotifier.NotifyPackageRevisionChange(watch.Modified, newPackage, metaPackage)
		}
	}

	for k, oldPackage := range oldPackageRevisions {
		metaPackage, found := existingPkgRevCRsMap[oldPackage.KubeObjectName()]
		if !found {
			klog.Warningf("no PackageRev CR found for PackageRevision %s", oldPackage.KubeObjectName())
		}
		if newPackageRevisionMap[k] == nil {
			r.objectNotifier.NotifyPackageRevisionChange(watch.Deleted, oldPackage, metaPackage)
		}
	}

	return newPackageMap, newPackageRevisionMap, nil
}
