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

package runtime

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLister struct {
	tags map[string][]string
	err  string
}

var _ TagLister = &fakeLister{}

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

func (frc *fakeLister) Name() string {
	return "fake"
}

func TestFilterParseSortTags(t *testing.T) {
	constraint, _ := semver.NewConstraint("*") // match everything
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
		"digests filtered out": {
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
			res := filterParseTags(tc.tags, constraint)
			sortVersions(res)
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
	const image = "ghcr.io/kptdev/krm-functions-catalog/test-function"
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
			expectedErr:   "no tag could be found matching the version constraint",
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
			expectedErr:   "no tag could be found matching the version constraint",
		},
		"digest replaced correctly": {
			functionImage: image + "@sha256:59a5a43c8fcafaf14b5fd4463ccb3fda61d6c0b55ff218cbb5783a29c8d6c20c",
			functionTag:   "~0.1",
			repoTags:      tagSet,
			expectedTag:   "v0.1.2",
		},
		// this case is technically impossible in a live scenario
		"image exist but has no tags": {
			functionImage: image,
			functionTag:   "~0.1",
			repoTags:      []string{},
			expectedErr:   "no tag could be found matching the version constraint",
		},
		"invalid image ref": {
			functionImage: "aaaaa~~234..=/",
			functionTag:   "~0.1",
			expectedErr:   "failed to parse image \"aaaaa~~234..=/\" as reference",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tr := &TagResolver{
				Listers: []TagLister{
					&fakeLister{
						err: tc.repoErr,
						tags: map[string][]string{
							image: tc.repoTags,
						},
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

func TestResolveFunctionImageMultipleListers(t *testing.T) {
	const image = "ghcr.io/kptdev/krm-functions-catalog/test-function"

	testCases := map[string]struct {
		listers     []TagLister
		functionTag string
		expectedTag string
		expectedErr string
	}{
		"combined tags from all listers": {
			listers: []TagLister{
				&fakeLister{tags: map[string][]string{image: {"v0.1.0", "v0.1.1"}}},
				&fakeLister{tags: map[string][]string{image: {"v0.2.0"}}},
			},
			functionTag: "~0.1",
			expectedTag: "v0.1.1",
		},
		"highest match from second lister": {
			listers: []TagLister{
				&fakeLister{tags: map[string][]string{image: {"v0.1.0"}}},
				&fakeLister{tags: map[string][]string{image: {"v0.2.0", "v0.2.1"}}},
			},
			functionTag: "*",
			expectedTag: "v0.2.1",
		},
		"continue after error": {
			listers: []TagLister{
				&fakeLister{err: "registry unavailable"},
				&fakeLister{tags: map[string][]string{image: {"v0.3.0"}}},
			},
			functionTag: "*",
			expectedTag: "v0.3.0",
		},
		"continues after empty list": {
			listers: []TagLister{
				&fakeLister{tags: map[string][]string{image: {}}},
				&fakeLister{tags: map[string][]string{image: {"v0.1.0", "v0.1.2"}}},
			},
			functionTag: "~0.1",
			expectedTag: "v0.1.2",
		},
		"no match in any lister": {
			listers: []TagLister{
				&fakeLister{tags: map[string][]string{image: {"v0.1.0"}}},
				&fakeLister{tags: map[string][]string{image: {"v0.2.0"}}},
			},
			functionTag: "0.3.x",
			expectedErr: "no tag could be found matching the version constraint",
		},
		"duplicate tags": {
			listers: []TagLister{
				&fakeLister{tags: map[string][]string{image: {"v0.1.0", "v0.2.0"}}},
				&fakeLister{tags: map[string][]string{image: {"v0.2.0", "v0.1.5"}}},
			},
			functionTag: "*",
			expectedTag: "v0.2.0",
		},
		"all listers fail": {
			listers: []TagLister{
				&fakeLister{err: "a down"},
				&fakeLister{err: "b down"},
			},
			functionTag: "*",
			expectedErr: "no tag could be found matching the version constraint",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tr := &TagResolver{Listers: tc.listers}
			resolvedImage, err := tr.ResolveFunctionImage(t.Context(), image, tc.functionTag)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, image+":"+tc.expectedTag, resolvedImage)
		})
	}
}

func TestAbbrevSlice(t *testing.T) {
	tagSet := []string{"v0.1", "v0.2.3", "v0.1.2", "v0", "v0.2", "v0.1.1", "v0.2.1", "v0.2.2"}
	var versions []*semver.Version
	for _, tag := range tagSet {
		versions = append(versions, semver.MustParse(tag))
	}

	testCases := []struct {
		input    []*semver.Version
		expected string
	}{
		{
			input:    []*semver.Version{},
			expected: "[]",
		},
		{
			input:    versions[:1],
			expected: "[v0.1]",
		},
		{
			input:    versions[:2],
			expected: "[v0.1, v0.2.3]",
		},
		{
			input:    versions[:3],
			expected: "[v0.1, v0.2.3, v0.1.2]",
		},
		{
			input:    versions[:4],
			expected: "[v0.1, v0.2.3, ..., v0]",
		},
		{
			input:    versions,
			expected: "[v0.1, v0.2.3, ..., v0.2.2]",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("len %d", len(tc.input)), func(t *testing.T) {
			result := abbrevSlice(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
