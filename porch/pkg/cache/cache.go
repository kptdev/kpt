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
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	kptoci "github.com/GoogleContainerTools/kpt/pkg/oci"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/pkg/meta"
	"github.com/GoogleContainerTools/kpt/porch/pkg/oci"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/watch"
)

// Cache allows us to keep state for repositories, rather than querying them every time.
//
// Cache Structure:
// <cacheDir>/git/
// * Caches bare git repositories in directories named based on the repository address.
// <cacheDir>/oci/
// * Caches oci images with further hierarchy underneath
// * We Cache image layers in <cacheDir>/oci/layers/ (this might be obsolete with the flattened Cache)
// * We Cache flattened tar files in <cacheDir>/oci/ (so we don't need to pull to read resources)
// * We poll the repositories (every minute) and Cache the discovered images in memory.
type Cache struct {
	mutex              sync.Mutex
	repositories       map[string]*cachedRepository
	cacheDir           string
	credentialResolver repository.CredentialResolver
	userInfoProvider   repository.UserInfoProvider
	metadataStore      meta.MetadataStore
	repoSyncFrequency  time.Duration
	objectNotifier     objectNotifier
}

type objectNotifier interface {
	NotifyPackageRevisionChange(eventType watch.EventType, obj repository.PackageRevision, objMeta meta.PackageRevisionMeta) int
}

type CacheOptions struct {
	CredentialResolver repository.CredentialResolver
	UserInfoProvider   repository.UserInfoProvider
	MetadataStore      meta.MetadataStore
	ObjectNotifier     objectNotifier
}

func NewCache(cacheDir string, repoSyncFrequency time.Duration, opts CacheOptions) *Cache {
	return &Cache{
		repositories:       make(map[string]*cachedRepository),
		cacheDir:           cacheDir,
		credentialResolver: opts.CredentialResolver,
		userInfoProvider:   opts.UserInfoProvider,
		metadataStore:      opts.MetadataStore,
		objectNotifier:     opts.ObjectNotifier,
		repoSyncFrequency:  repoSyncFrequency,
	}
}

func (c *Cache) OpenRepository(ctx context.Context, repositorySpec *configapi.Repository) (*cachedRepository, error) {
	ctx, span := tracer.Start(ctx, "Cache::OpenRepository", trace.WithAttributes())
	defer span.End()

	switch repositoryType := repositorySpec.Spec.Type; repositoryType {
	case configapi.RepositoryTypeOCI:
		ociSpec := repositorySpec.Spec.Oci
		if ociSpec == nil {
			return nil, fmt.Errorf("oci not configured")
		}
		key := "oci://" + ociSpec.Registry
		c.mutex.Lock()
		defer c.mutex.Unlock()

		cr := c.repositories[key]

		if cr == nil {
			cacheDir := filepath.Join(c.cacheDir, "oci")
			storage, err := kptoci.NewStorage(cacheDir)
			if err != nil {
				return nil, err
			}

			r, err := oci.OpenRepository(repositorySpec.Name, repositorySpec.Namespace, repositorySpec.Spec.Content, ociSpec, repositorySpec.Spec.Deployment, storage)
			if err != nil {
				return nil, err
			}
			cr = newRepository(key, repositorySpec, r, c.objectNotifier, c.metadataStore, c.repoSyncFrequency)
			c.repositories[key] = cr
		}
		return cr, nil

	case configapi.RepositoryTypeGit:
		gitSpec := repositorySpec.Spec.Git
		if gitSpec == nil {
			return nil, errors.New("git property is required")
		}
		if gitSpec.Repo == "" {
			return nil, errors.New("git.repo property is required")
		}
		if !isPackageContent(repositorySpec.Spec.Content) {
			return nil, fmt.Errorf("git repository supports Package content only; got %q", string(repositorySpec.Spec.Content))
		}
		key := "git://" + gitSpec.Repo + gitSpec.Directory

		c.mutex.Lock()
		defer c.mutex.Unlock()

		cr := c.repositories[key]
		if cr == nil {
			var mbs git.MainBranchStrategy
			if gitSpec.CreateBranch {
				mbs = git.CreateIfMissing
			} else {
				mbs = git.ErrorIfMissing
			}
			if r, err := git.OpenRepository(ctx, repositorySpec.Name, repositorySpec.Namespace, gitSpec, repositorySpec.Spec.Deployment, filepath.Join(c.cacheDir, "git"), git.GitRepositoryOptions{
				CredentialResolver: c.credentialResolver,
				UserInfoProvider:   c.userInfoProvider,
				MainBranchStrategy: mbs,
			}); err != nil {
				return nil, err
			} else {
				cr = newRepository(key, repositorySpec, r, c.objectNotifier, c.metadataStore, c.repoSyncFrequency)
				c.repositories[key] = cr
			}
		} else {
			// If there is an error from the background refresh goroutine, return it.
			if err := cr.getRefreshError(); err != nil {
				return nil, err
			}
		}
		return cr, nil

	default:
		return nil, fmt.Errorf("type %q not supported", repositoryType)
	}
}

func isPackageContent(content configapi.RepositoryContent) bool {
	return content == configapi.RepositoryContentPackage
}

func (c *Cache) CloseRepository(repositorySpec *configapi.Repository) error {
	var key string

	switch repositorySpec.Spec.Type {
	case configapi.RepositoryTypeOCI:
		oci := repositorySpec.Spec.Oci
		if oci == nil {
			return fmt.Errorf("oci not configured for %s:%s", repositorySpec.ObjectMeta.Namespace, repositorySpec.ObjectMeta.Name)
		}
		key = "oci://" + oci.Registry

	case configapi.RepositoryTypeGit:
		git := repositorySpec.Spec.Git
		if git == nil {
			return fmt.Errorf("git not configured for %s:%s", repositorySpec.ObjectMeta.Namespace, repositorySpec.ObjectMeta.Name)
		}
		key = "git://" + git.Repo + git.Directory

	default:
		return fmt.Errorf("unknown repository type: %q", repositorySpec.Spec.Type)
	}

	// TODO: Multiple Repository resources can point to the same underlying repository
	// and therefore the same cache. Implement reference counting

	var repository *cachedRepository
	{
		c.mutex.Lock()
		if r, ok := c.repositories[key]; ok {
			delete(c.repositories, key)
			repository = r
		}
		c.mutex.Unlock()
	}

	if repository != nil {
		return repository.Close()
	} else {
		return nil
	}
}
