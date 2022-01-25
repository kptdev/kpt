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
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

var tracer = otel.Tracer("cache")

type cachedRepository struct {
	id     string
	repo   repository.Repository
	cancel context.CancelFunc

	mutex          sync.Mutex
	cachedPackages []repository.PackageRevision
	// TODO: Currently we support repositories with homogenous content (only packages xor functions). Model this more optimally?
	cachedFunctions []repository.Function
}

func newRepository(id string, repo repository.Repository) *cachedRepository {
	ctx, cancel := context.WithCancel(context.Background())
	r := &cachedRepository{
		id:     id,
		repo:   repo,
		cancel: cancel,
	}

	go r.pollForever(ctx)

	return r
}

var _ repository.Repository = &cachedRepository{}
var _ repository.FunctionRepository = &cachedRepository{}

func (r *cachedRepository) ListPackageRevisions(ctx context.Context) ([]repository.PackageRevision, error) {
	packages, err := r.getPackages(ctx, false)
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

func (r *cachedRepository) getPackages(ctx context.Context, forceRefresh bool) ([]repository.PackageRevision, error) {
	r.mutex.Lock()
	packages := r.cachedPackages
	r.mutex.Unlock()

	if forceRefresh {
		packages = nil
	}

	if packages == nil {
		// TODO: Avoid simultaneous fetches?
		p, err := r.repo.ListPackageRevisions(ctx)
		if err != nil {
			return nil, err
		}
		packages = p

		r.mutex.Lock()
		r.cachedPackages = p
		r.mutex.Unlock()
	}

	return packages, nil
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

func (r *cachedRepository) UpdatePackage(ctx context.Context, old repository.PackageRevision) (repository.PackageDraft, error) {
	created, err := r.repo.UpdatePackage(ctx, old)
	if err != nil {
		return nil, err
	}

	return &cachedDraft{
		PackageDraft: created,
		cache:        r,
	}, nil
}

func (r *cachedRepository) update(closed repository.PackageRevision) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for i, cached := range r.cachedPackages {
		if cached.Name() == closed.Name() {
			if cached == closed {
				return
			}
			// TODO: more sophisticated conflict reconciliation?
			r.cachedPackages[i] = closed
			return
		}
	}

	r.cachedPackages = append(r.cachedPackages, closed)
}

func (r *cachedRepository) DeletePackageRevision(ctx context.Context, old repository.PackageRevision) error {
	if err := r.repo.DeletePackageRevision(ctx, old); err != nil {
		return err
	}

	r.mutex.Lock()
	// TODO: Do something more efficient than a full cache flush
	r.cachedPackages = nil
	r.mutex.Unlock()

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
	ctx, span := tracer.Start(ctx, "Repository.pollOnce", trace.WithAttributes())
	defer span.End()

	if _, err := r.getPackages(ctx, true); err != nil {
		klog.Warningf("error polling repo packages %s: %v", r.id, err)
	}
	if _, err := r.getFunctions(ctx, true); err != nil {
		klog.Warningf("error polling repo functions %s: %v", r.id, err)
	}
}
