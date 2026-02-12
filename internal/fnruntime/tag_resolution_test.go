// Copyright 2026 The kpt Authors
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

package fnruntime

import (
	"context"
	"errors"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	tags map[string][]string
	err  string
}

func (frc *fakeLister) List(_ context.Context, image string) ([]string, error) {
	if frc.err != "" {
		return nil, errors.New(frc.err)
	}

	if frc.tags == nil {
		return []string{}, nil
	}

	tags, ok := frc.tags[image]
	if !ok {
		return []string{}, nil
	}

	return tags, nil
}

var _ TagLister = &fakeLister{}

func TestFilterParseSortTags(t *testing.T) {
	testCases := map[string]struct {
		tags     []string
		expected []string
	}{
		"correct sorting": {
			tags:     []string{"v0.1.0", "v0.1.2", "v0.1.1"},
			expected: []string{"v0.1.2", "v0.1.1", "v0.1.0"},
		},
		"correct sorting 2": {
			tags:     []string{"v0.1", "v0.2.1", "v0.1.2", "v0.2", "v0.1.1", "v0.2.3", "v0.2.2", "v0"},
			expected: []string{"v0.2.3", "v0.2.2", "v0.2.1", "v0.2", "v0.1.2", "v0.1.1", "v0.1", "v0"},
		},
		"sha hashes filtered out": {
			tags:     []string{"v0.1.0", "sha256-59a5a43c8fcafaf14b5fd4463ccb3fda61d6c0b55ff218cbb5783a29c8d6c20c.sbom", "v0.1.1"},
			expected: []string{"v0.1.1", "v0.1.0"},
		},
		"common pattern filtered out": {
			tags:     []string{"v0.1.0", "master-git-38f885f", "v0.1.1"},
			expected: []string{"v0.1.1", "v0.1.0"},
		},
		"short hash filtered out": {
			tags:     []string{"v0.1.0", "38f885f", "v0.1.1"},
			expected: []string{"v0.1.1", "v0.1.0"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			res := filterParseSortTags(tc.tags)
			assert.Equal(t, tc.expected, stringifyVersionSlice(res))
		})
	}
}

func stringifyVersionSlice(list []*semver.Version) []string {
	var output []string
	for _, version := range list {
		output = append(output, version.Original())
	}
	return output
}

func TestResolveFunctionImage(t *testing.T) {
	const image = "test-function"
	tagSet := []string{"v0.1", "v0.2.3", "v0.1.2", "v0", "v0.2", "v0.1.1", "v0.2.1", "v0.2.2"}

	testCases := map[string]struct {
		functionTag   string
		functionImage string
		repoTags      []string
		repoErr       string

		expectedTag string
		expectedErr string
	}{
		"basic tilde": {
			functionImage: image,
			functionTag:   "~0.1",
			repoTags:      tagSet,
			expectedTag:   "v0.1.2",
		},
		"basic range": {
			functionImage: image,
			functionTag:   "0.2 - 0.2.2",
			repoTags:      tagSet,
			expectedTag:   "v0.2.2",
		},
		"range with no latest": {
			functionImage: image,
			functionTag:   "0.2 - 0.2.2",
			repoTags:      tagSet[:len(tagSet)-1],
			expectedTag:   "v0.2.1",
		},
		"no matching": {
			functionImage: image,
			functionTag:   "0.3.x",
			repoTags:      tagSet,
			expectedErr:   "no remote tag matched the version constraint",
		},
		"image preserved on empty tag": {
			functionImage: image + ":v0.3.1",
			repoTags:      tagSet,
			expectedTag:   "v0.3.1",
		},
		"exact semver tag": {
			functionImage: image,
			functionTag:   "v0.3.1",
			repoTags:      tagSet,
			expectedTag:   "v0.3.1",
		},
		"no listing with exact semver tag": {
			functionImage: image,
			functionTag:   "v0.3.1",
			repoErr:       "test",
			expectedTag:   "v0.3.1",
		},
		"exact non-semver tag": {
			functionImage: image,
			functionTag:   "master-git-38f885f",
			repoTags:      tagSet,
			expectedTag:   "master-git-38f885f",
		},
		"no listing with exact non-semver tag": {
			functionImage: image,
			functionTag:   "master-git-38f885f",
			repoErr:       "test",
			expectedTag:   "master-git-38f885f",
		},
		"list failure": {
			functionImage: image,
			functionTag:   "~0.1",
			repoErr:       "test",
			expectedErr:   "failed to list tags for image",
		},
		"sha replaced correctly": {
			functionImage: image + "@sha256:59a5a43c8fcafaf14b5fd4463ccb3fda61d6c0b55ff218cbb5783a29c8d6c20c",
			functionTag:   "~0.1",
			repoTags:      tagSet,
			expectedTag:   "v0.1.2",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tr := &TagResolver{
				lister: &fakeLister{
					err: tc.repoErr,
					tags: map[string][]string{
						image: tc.repoTags,
					},
				},
			}
			resolvedImage, err := tr.ResolveFunctionImage(t.Context(), tc.functionImage, tc.functionTag)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, image+":"+tc.expectedTag, resolvedImage)
		})
	}
}
