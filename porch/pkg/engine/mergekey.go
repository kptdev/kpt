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

package engine

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/util/addmergecomment"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// addMergeKeyMutation adds merge-key comment directive to reconcile
// identity of resources in a downstream package with the ones in upstream package
// This is required to ensure package update is able to merge resources in
// downstream package with upstream.
func ensureMergeKey(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, error) {
	pr := &packageReader{
		input: resources,
		extra: map[string]string{},
	}

	result := repository.PackageResources{
		Contents: map[string]string{},
	}

	amc := &addmergecomment.AddMergeComment{}

	pipeline := kio.Pipeline{
		Inputs:  []kio.Reader{pr},
		Filters: []kio.Filter{kio.FilterAll(amc)},
		Outputs: []kio.Writer{&packageWriter{
			output: result,
		}},
	}

	if err := pipeline.Execute(); err != nil {
		return repository.PackageResources{}, fmt.Errorf("failed to add merge-key directive: %w", err)
	}

	for k, v := range pr.extra {
		result.Contents[k] = v
	}

	return result, nil
}
