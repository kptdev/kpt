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

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
)

type cachedDraft struct {
	repository.PackageDraft
	cache *cachedRepository
}

var _ repository.PackageDraft = &cachedDraft{}

func (cd *cachedDraft) Close(ctx context.Context) (repository.PackageRevision, error) {
	closed, err := cd.PackageDraft.Close(ctx)
	if err != nil {
		return nil, err
	}

	err = cd.cache.reconcileCache(ctx, "close-draft")
	if err != nil {
		return nil, err
	}

	cpr := cd.cache.getPackageRevision(closed.Key())
	if cpr == nil {
		return nil, fmt.Errorf("closed draft not found")
	}

	return cpr, nil
}
