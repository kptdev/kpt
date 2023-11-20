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

package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
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

	lastVersion string

	// We use separate mutexes for cache map changes and for the overall
	// reconcile process. We want update, delete, and reconcile
	// to all block on the reconcileMutex, which could be held for a long time
	// during reconcile. For much of that time (during reconcile) we do NOT
	// want to block reads. There are a few protected areas where we touch map
	// entries where we need to block reads, so those will also grab the general
	// mutex.
	//
	// Any code that needs to hold both locks MUST get the reconcileMutex first,
	// or we could end up with deadlocks
	mutex                  sync.RWMutex
	reconcileMutex         sync.Mutex
	cachedPackageRevisions map[repository.PackageRevisionKey]*cachedPackageRevision

	// TODO: Currently we support repositories with homogenous content (only packages xor functions). Model this more optimally?
	cachedFunctions []repository.Function
	// Error encountered on repository refresh by the refresh goroutine.
	// This is returned back by the cache to the background goroutine when it calls periodicall to resync repositories.
	refreshRevisionsError error

	objectNotifier objectNotifier

	metadataStore meta.MetadataStore
}

func newRepository(id string, repoSpec *configapi.Repository, repo repository.Repository, objectNotifier objectNotifier, metadataStore meta.MetadataStore, repoSyncFrequency time.Duration) *cachedRepository {
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

	go r.pollForever(ctx, repoSyncFrequency)

	return r
}

func (r *cachedRepository) Version(ctx context.Context) (string, error) {
	return r.repo.Version(ctx)
}

func (r *cachedRepository) ListPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	packages, err := r.getPackageRevisions(ctx, filter)
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

func (r *cachedRepository) getPackageRevisions(ctx context.Context, filter repository.ListPackageRevisionFilter) ([]repository.PackageRevision, error) {
	packageRevisions, err := r.getCachedPackageRevisions(ctx)
	if err != nil {
		return nil, err
	}

	return toPackageRevisionSlice(packageRevisions, filter), nil
}

// getCachedPackageRevisions returns the cache contents, blocking until
// the cache is loaded
// caller must NOT hold the lock
// returned *map* is a copy and can be operated on without locks
// map entries are NOT copies and should not be modified
func (r *cachedRepository) getCachedPackageRevisions(ctx context.Context) (map[repository.PackageRevisionKey]*cachedPackageRevision, error) {
	err := r.blockUntilLoaded(ctx)
	if err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	packageRevisions := make(map[repository.PackageRevisionKey]*cachedPackageRevision, len(r.cachedPackageRevisions))
	for k, v := range r.cachedPackageRevisions {
		packageRevisions[k] = v
	}

	return packageRevisions, r.refreshRevisionsError
}

// blocks waiting until the cache is loaded
func (r *cachedRepository) blockUntilLoaded(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("repo %s: stopped waiting for load because context is done: %v", r.id, ctx.Err())
		default:
			r.mutex.RLock()
			if r.cachedPackageRevisions != nil {
				r.mutex.RUnlock()
				return nil
			}
			r.mutex.RUnlock()
			time.Sleep(1 * time.Second)
		}
	}
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
	err := r.blockUntilLoaded(ctx)
	if err != nil {
		return nil, err
	}

	// we need the reconcileMutex, get it first
	r.reconcileMutex.Lock()
	defer r.reconcileMutex.Unlock()

	// we will also need the cache lock
	r.mutex.Lock()
	defer r.mutex.Unlock()

	k := updated.Key()
	// previous := r.cachedPackageRevisions[k]

	if v1alpha1.LifecycleIsPublished(updated.Lifecycle()) {
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

	return cached, nil
}

func (r *cachedRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	// get the reconcile lock first, before touching the underlying repo
	r.reconcileMutex.Lock()
	defer r.reconcileMutex.Unlock()

	// Unwrap
	unwrapped := old.(*cachedPackageRevision).PackageRevision
	if err := r.repo.DeletePackageRevision(ctx, unwrapped); err != nil {
		return err
	}

	r.mutex.Lock()
	if r.cachedPackageRevisions != nil {
		k := old.Key()
		delete(r.cachedPackageRevisions, k)

		// Recompute latest package revisions.
		// TODO: Only for affected object / key?
		identifyLatestRevisions(r.cachedPackageRevisions)
	}
	r.mutex.Unlock()

	return nil
}

func (r *cachedRepository) ListPackages(ctx context.Context, filter repository.ListPackageFilter) ([]repository.Package, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *cachedRepository) CreatePackage(ctx context.Context, obj *v1alpha1.Package) (repository.Package, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *cachedRepository) DeletePackage(ctx context.Context, old repository.Package) error {
	return fmt.Errorf("not implemented")
}

func (r *cachedRepository) Close() error {
	r.cancel()

	r.reconcileMutex.Lock()
	defer r.reconcileMutex.Unlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Make sure that watch events are sent for packagerevisions that are
	// removed as part of closing the repository.
	sent := 0
	for _, pr := range r.cachedPackageRevisions {
		nn := types.NamespacedName{
			Name:      pr.KubeObjectName(),
			Namespace: pr.KubeObjectNamespace(),
		}
		// There isn't really any correct way to handle finalizers here. We are removing
		// the repository, so we have to just delete the PackageRevision regardless of any
		// finalizers.
		klog.Infof("repo %s: deleting packagerev %s/%s because repository is closed", r.id, nn.Namespace, nn.Name)
		pkgRevMeta, err := r.metadataStore.Delete(context.TODO(), nn, true)
		if err != nil {
			// There isn't much use in returning an error here, so we just log it
			// and create a PackageRevisionMeta with just name and namespace. This
			// makes sure that the Delete event is sent.
			klog.Warningf("Error looking up PackageRev CR for %s: %v")
			pkgRevMeta = meta.PackageRevisionMeta{
				Name:      nn.Name,
				Namespace: nn.Namespace,
			}
		}
		sent += r.objectNotifier.NotifyPackageRevisionChange(watch.Deleted, pr, pkgRevMeta)
	}
	klog.Infof("repo %s: sent %d notifications for %d package revisions during close", r.id, sent, len(r.cachedPackageRevisions))
	return r.repo.Close()
}

// pollForever will continue polling until signal channel is closed or ctx is done.
func (r *cachedRepository) pollForever(ctx context.Context, repoSyncFrequency time.Duration) {
	r.pollOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			klog.V(2).Infof("repo %s: exiting repository poller, because context is done: %v", r.id, ctx.Err())
			return
		default:
			r.pollOnce(ctx)
			time.Sleep(repoSyncFrequency)
		}
	}
}

func (r *cachedRepository) pollOnce(ctx context.Context) {
	start := time.Now()
	klog.Infof("repo %s: poll started", r.id)
	defer func() { klog.Infof("repo %s: poll finished in %f secs", r.id, time.Since(start).Seconds()) }()
	ctx, span := tracer.Start(ctx, "Repository::pollOnce", trace.WithAttributes())
	defer span.End()

	if err := r.reconcileCache(ctx); err != nil {
		klog.Warningf("error polling repo packages %s: %v", r.id, err)
	}
	if _, err := r.getFunctions(ctx, true); err != nil {
		klog.Warningf("error polling repo functions %s: %v", r.id, err)
	}
}

// reconcileCache updates the cached map for this repository
// it also triggers notifications for all package changes
// caller must NOT hold any locks
func (r *cachedRepository) reconcileCache(ctx context.Context) error {
	// TODO: Avoid simultaneous fetches?
	// TODO: Push-down partial refresh?

	start := time.Now()
	defer func() { klog.Infof("repo %s: refresh finished in %f secs", r.id, time.Since(start).Seconds()) }()

	curVer, err := r.Version(ctx)
	if err != nil {
		return err
	}

	if curVer == r.lastVersion {
		return nil
	}

	// get the reconcile lock first, to block any repo-level mutations
	r.reconcileMutex.Lock()
	defer r.reconcileMutex.Unlock()

	if gitRepo, isGitRepo := r.repo.(git.GitRepository); isGitRepo {
		// TODO: Figure out a way to do this without the cache layer
		//  needing to know what type of repo we are working with.
		if err := gitRepo.UpdateDeletionProposedCache(); err != nil {
			return err
		}
	}

	// Look up all existing PackageRevCRs so we an compare those to the
	// actual Packagerevisions found in git/oci, and add/prune PackageRevCRs
	// as necessary.
	existingPkgRevCRs, err := r.metadataStore.List(ctx, r.repoSpec)
	if err != nil {
		return err
	}
	// Create a map so we can quickly check if a specific PackageRevisionMeta exists.
	existingPkgRevCRsMap := make(map[string]meta.PackageRevisionMeta)
	for i := range existingPkgRevCRs {
		pr := existingPkgRevCRs[i]
		existingPkgRevCRsMap[pr.Name] = pr
	}

	newPackageRevisions, err := r.repo.ListPackageRevisions(ctx, repository.ListPackageRevisionFilter{})
	if err != nil {
		return fmt.Errorf("error listing packages: %w", err)
	}

	// Build mapping from kubeObjectName to PackageRevisions for new PackageRevisions.
	newPackageRevisionNames := make(map[string]*cachedPackageRevision, len(newPackageRevisions))
	for _, newPackage := range newPackageRevisions {
		kname := newPackage.KubeObjectName()
		if newPackageRevisionNames[kname] != nil {
			klog.Warningf("repo %s: found duplicate packages with name %v", kname)
		}

		pkgRev := &cachedPackageRevision{
			PackageRevision:  newPackage,
			isLatestRevision: false,
		}
		newPackageRevisionNames[newPackage.KubeObjectName()] = pkgRev
	}

	// Build mapping from kubeObjectName to PackageRevisions for existing PackageRevisions
	// Grab the RLock while we create this map
	r.mutex.RLock()
	oldPackageRevisionNames := make(map[string]*cachedPackageRevision, len(r.cachedPackageRevisions))
	for _, oldPackage := range r.cachedPackageRevisions {
		oldPackageRevisionNames[oldPackage.KubeObjectName()] = oldPackage
	}
	r.mutex.RUnlock()

	// We go through all PackageRev CRs that represents PackageRevisions
	// in the current repo and make sure they all have a corresponding
	// PackageRevision. The ones that doesn't is removed.
	for _, prm := range existingPkgRevCRs {
		if _, found := newPackageRevisionNames[prm.Name]; !found {
			klog.Infof("repo %s: deleting PackageRev %s/%s because parent PackageRevision was not found",
				r.id, prm.Namespace, prm.Name)
			if _, err := r.metadataStore.Delete(ctx, types.NamespacedName{
				Name:      prm.Name,
				Namespace: prm.Namespace,
			}, true); err != nil {
				if !apierrors.IsNotFound(err) {
					// This will be retried the next time the sync runs.
					klog.Warningf("repo %s: unable to delete PackageRev CR for %s/%s: %w",
						r.id, prm.Name, prm.Namespace, err)
				}
			}
		}
	}

	// We go through all the PackageRevisions and make sure they have
	// a corresponding PackageRev CR.
	for pkgRevName, pkgRev := range newPackageRevisionNames {
		if _, found := existingPkgRevCRsMap[pkgRevName]; !found {
			pkgRevMeta := meta.PackageRevisionMeta{
				Name:      pkgRevName,
				Namespace: r.repoSpec.Namespace,
			}
			if _, err := r.metadataStore.Create(ctx, pkgRevMeta, r.repoSpec.Name, pkgRev.UID()); err != nil {
				// TODO: We should try to find a way to make these errors available through
				// either the repository CR or the PackageRevision CR. This will be
				// retried on the next sync.
				klog.Warningf("unable to create PackageRev CR for %s/%s: %w",
					r.repoSpec.Namespace, pkgRevName, err)
			}
		}
	}

	// fix up the isLatestRevision in the new maps
	newPackageRevisionMap := make(map[repository.PackageRevisionKey]*cachedPackageRevision, len(newPackageRevisions))
	for _, newPackage := range newPackageRevisions {
		k := newPackage.Key()
		pkgRev := &cachedPackageRevision{
			PackageRevision:  newPackage,
			isLatestRevision: false,
		}
		newPackageRevisionMap[k] = pkgRev
	}

	identifyLatestRevisions(newPackageRevisionMap)

	// hold the RW lock while swap in the new packages
	// we do this now, *before* sending notifications, so that
	// anyone responding to the notification will get the new values
	r.mutex.Lock()
	r.cachedPackageRevisions = newPackageRevisionMap
	r.lastVersion = curVer
	r.mutex.Unlock()

	// Send notification for packages that changed.
	addSent := 0
	modSent := 0
	for kname, newPackage := range newPackageRevisionNames {
		oldPackage := oldPackageRevisionNames[kname]
		metaPackage, found := existingPkgRevCRsMap[newPackage.KubeObjectName()]
		if !found {
			klog.Warningf("no PackageRev CR found for PackageRevision %s", newPackage.KubeObjectName())
		}
		if oldPackage == nil {
			addSent += r.objectNotifier.NotifyPackageRevisionChange(watch.Added, newPackage, metaPackage)
		} else {
			if oldPackage.ResourceVersion() != newPackage.ResourceVersion() {
				modSent += r.objectNotifier.NotifyPackageRevisionChange(watch.Modified, newPackage, metaPackage)
			}
		}
	}

	delSent := 0
	// Send notifications for packages that was deleted in the SoT
	for kname, oldPackage := range oldPackageRevisionNames {
		if newPackageRevisionNames[kname] == nil {
			nn := types.NamespacedName{
				Name:      oldPackage.KubeObjectName(),
				Namespace: oldPackage.KubeObjectNamespace(),
			}
			klog.Infof("repo %s: deleting PackageRev %s/%s because PackageRevision was removed from SoT",
				r.id, nn.Namespace, nn.Name)
			metaPackage, err := r.metadataStore.Delete(ctx, nn, true)
			if err != nil {
				if !apierrors.IsNotFound(err) {
					klog.Warningf("repo %s: error deleting PkgRevMeta %s: %v", r.id, nn, err)
				}
				metaPackage = meta.PackageRevisionMeta{
					Name:      nn.Name,
					Namespace: nn.Namespace,
				}
			}
			delSent += r.objectNotifier.NotifyPackageRevisionChange(watch.Deleted, oldPackage, metaPackage)
		}
	}
	klog.Infof("repo %s: addSent %d, modSent %d, delSent for %d old and %d new repo packages", r.id, addSent, modSent, len(oldPackageRevisionNames), len(newPackageRevisionNames))
	return nil
}
