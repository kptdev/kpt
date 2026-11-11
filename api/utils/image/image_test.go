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

package image

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBestSemverMatch(t *testing.T) {
	testCases := map[string]struct {
		constraint  string
		tags        []string
		expected    string
		expectedErr string
	}{
		"selects highest matching version": {
			constraint: ">= 0.4.0 < 0.5.0",
			tags: []string{
				"v0.4.1",
				"v0.4",
				"@sha256:abcdef123456",
			},
			expected: "v0.4.1",
		},
		"exact version match": {
			constraint: "0.1.1",
			tags: []string{
				"v0.1.1",
				"v0.1",
			},
			expected: "v0.1.1",
		},
		"no matching version for valid constraint": {
			constraint: "> 1.0.0",
			tags: []string{
				"v0.1.1",
				"v0.1",
			},
			expectedErr: "no tag matching",
		},
		"invalid semver constraint": {
			constraint: ">> 1.0.0",
			tags: []string{
				"v1.1.0",
			},
			expectedErr: "invalid semver constraint",
		},
		"skips sha256-tagged entries": {
			constraint: ">= 0.4.0",
			tags: []string{
				"v0.4.1",
				"v0.4",
				"@sha256:abcdef123456",
			},
			expected: "v0.4.1",
		},
		"matches without registry prefix": {
			constraint: ">= 0.4.0",
			tags: []string{
				"v0.4.1",
				"v0.4",
				"@sha256:abcdef123456",
			},
			expected: "v0.4.1",
		},
		"empty cache keys": {
			constraint:  ">= 0.1.0",
			tags:        []string{},
			expectedErr: "no tag matching",
		},
		"selects greatest from multiple matches": {
			constraint: ">= 1.0.0 < 2.0.0",
			tags: []string{
				"v1.0.0",
				"v1.1.0",
				"v1.2.0",
				"v2.0.0",
			},
			expected: "v1.2.0",
		},
		"skips entries with unparseable versions": {
			constraint: ">= 1.0.0",
			tags: []string{
				"v1.0.0",
				"vnotaversion",
			},
			expected: "v1.0.0",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			best, err := FindBestSemverMatch(tc.constraint, tc.tags)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, best)
			}
		})
	}
}

func TestMatchesAnyConstraint(t *testing.T) {
	testCases := map[string]struct {
		version     string
		constraints []string
		expected    bool
	}{
		"empty constraints matches nothing": {
			version:     "v0.4.1",
			constraints: []string{},
			expected:    false,
		},
		"empty version matches everything": {
			version:     "",
			constraints: []string{">= v0.4.0"},
			expected:    true,
		},
		"empty version with empty constraints matches nothing": {
			version:     "",
			constraints: []string{},
			expected:    false,
		},
		"wildcard matches any version": {
			version:     "v0.4.1",
			constraints: []string{"*"},
			expected:    true,
		},
		"exact version constraint matches": {
			version:     "v0.4.1",
			constraints: []string{"v0.4.1"},
			expected:    true,
		},
		"range constraint matches in-range version": {
			version:     "v0.4.2",
			constraints: []string{">= v0.4.0 < v0.5.0"},
			expected:    true,
		},
		"range constraint rejects out-of-range version": {
			version:     "v0.5.0",
			constraints: []string{">= v0.4.0 < v0.5.0"},
			expected:    false,
		},
		"matches first of multiple constraints": {
			version:     "v0.4.2",
			constraints: []string{">= v0.4.0 < v0.5.0", ">= v1.0.0"},
			expected:    true,
		},
		"matches second of multiple constraints": {
			version:     "v1.2.0",
			constraints: []string{">= v0.4.0 < v0.5.0", ">= v1.0.0"},
			expected:    true,
		},
		"no matching constraint": {
			version:     "v0.3.9",
			constraints: []string{">= v0.4.0 < v0.5.0"},
			expected:    false,
		},
		"non-semver version falls back to exact match hit": {
			version:     "latest",
			constraints: []string{"latest"},
			expected:    true,
		},
		"non-semver version falls back to exact match miss": {
			version:     "latest",
			constraints: []string{"stable"},
			expected:    false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := MatchesAnyConstraint(tc.version, tc.constraints)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestImageParse(t *testing.T) {
	const (
		registry         = "ghcr.io"
		registryWithPort = "my-registry.com:5000"
		subpath          = "kptdev/krm-functions-catalog"
		image            = "apply-setters"
		tag              = "v0.2.3"
		digest           = "sha256:7d89a74f106241391f687fc2985c8e6de597bb21f0d0014def5edc730618d9cc"
	)

	testCases := map[string]struct {
		input string
		want  ParsedImage
	}{
		"empty": {
			input: "",
			want:  ParsedImage{},
		},
		"base name only": {
			input: image,
			want: ParsedImage{
				Original: image,
				BaseName: image,
			},
		},
		"tag only": {
			input: fmt.Sprintf("%s:%s", image, tag),
			want: ParsedImage{
				Original: fmt.Sprintf("%s:%s", image, tag),
				BaseName: image,
				Tag:      tag,
			},
		},
		"registry no path": {
			input: fmt.Sprintf("%s/%s", registry, image),
			want: ParsedImage{
				Original: fmt.Sprintf("%s/%s", registry, image),
				Registry: registry,
				BaseName: image,
			},
		},
		"registry with path": {
			input: fmt.Sprintf("%s/%s:%s", subpath, image, tag),
			want: ParsedImage{
				Original: fmt.Sprintf("%s/%s:%s", subpath, image, tag),
				SubPath:  subpath,
				BaseName: image,
				Tag:      tag,
			},
		},
		"fully qualified, no digest": {
			input: fmt.Sprintf("%s/%s/%s:%s", registry, subpath, image, tag),
			want: ParsedImage{
				Original: fmt.Sprintf("%s/%s/%s:%s", registry, subpath, image, tag),
				Registry: registry,
				SubPath:  subpath,
				BaseName: image,
				Tag:      tag,
			},
		},
		"digest without tag": {
			input: fmt.Sprintf("%s@%s", image, digest),
			want: ParsedImage{
				Original: fmt.Sprintf("%s@%s", image, digest),
				BaseName: image,
				Digest:   digest,
			},
		},
		"tag and digest": {
			input: fmt.Sprintf("%s:%s@%s", image, tag, digest),
			want: ParsedImage{
				Original: fmt.Sprintf("%s:%s@%s", image, tag, digest),
				BaseName: image,
				Tag:      tag,
				Digest:   digest,
			},
		},
		"registry with port": {
			input: fmt.Sprintf("%s/%s/%s:%s", registryWithPort, subpath, image, tag),
			want: ParsedImage{
				Original: fmt.Sprintf("%s/%s/%s:%s", registryWithPort, subpath, image, tag),
				Registry: registryWithPort,
				SubPath:  subpath,
				BaseName: image,
				Tag:      tag,
			},
		},
		"digest with nested repository": {
			input: fmt.Sprintf("%s/%s/%s@%s", registry, subpath, image, digest),
			want: ParsedImage{
				Original: fmt.Sprintf("%s/%s/%s@%s", registry, subpath, image, digest),
				Registry: registry,
				SubPath:  subpath,
				BaseName: image,
				Digest:   digest,
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := Parse(tc.input)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, strings.TrimSuffix(strings.TrimSuffix(got.Full(), "@"+got.Digest), ":"+got.Tag), got.Repository())
		})
	}
}

func TestWithPrefixLeavesAReferenceUnchanged(t *testing.T) {
	const prefix = "ghcr.io/kptdev/krm-functions-catalog"
	assert.Equal(t, prefix+"/apply-setters", WithPrefix("apply-setters", prefix))
	assert.Equal(t, "example.com/apply-setters:v1", WithPrefix("example.com/apply-setters:v1", prefix))
	assert.Equal(t, "apply-setters", WithPrefix("apply-setters", ""))
}
