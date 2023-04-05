// Copyright 2021 The kpt Authors
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

package parse

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_pkgURLFromGHURL(t *testing.T) {
	tests := []struct {
		name     string
		ghURL    string
		want     string
		errS     string
		branches []string
	}{
		{
			name:     "simple",
			ghURL:    "https://github.com/owner/repo",
			want:     "https://github.com/owner/repo.git",
			errS:     "",
			branches: nil,
		},
		{
			name:     "with trailing slash",
			ghURL:    "https://github.com/owner/repo/",
			want:     "https://github.com/owner/repo.git",
			errS:     "",
			branches: nil,
		},
		{
			name:     "with ref",
			ghURL:    "https://github.com/owner/repo/tree/main",
			want:     "https://github.com/owner/repo.git@main",
			errS:     "",
			branches: nil,
		},
		{
			name:     "with commit SHA",
			ghURL:    "https://github.com/owner/repo/tree/fc0193e5cf7ff836f8208644ff2dc14901ed06c9",
			want:     "https://github.com/owner/repo.git@fc0193e5cf7ff836f8208644ff2dc14901ed06c9",
			errS:     "",
			branches: nil,
		},
		{
			name:     "with ref with branches",
			ghURL:    "https://github.com/owner/repo/tree/main",
			want:     "https://github.com/owner/repo.git@main",
			errS:     "",
			branches: []string{"test", "main"},
		},
		{
			name:  "with commit SHA with branches",
			ghURL: "https://github.com/owner/repo/tree/fc0193e5cf7ff836f8208644ff2dc14901ed06c9",
			want:  "https://github.com/owner/repo.git@fc0193e5cf7ff836f8208644ff2dc14901ed06c9", errS: "", branches: []string{"test", "main"},
		},
		{
			name:     "with ref with branches ambiguous",
			ghURL:    "https://github.com/owner/repo/tree/main/foo/bar",
			want:     "",
			errS:     "ambiguous repo/dir@version specify '.git' in argument",
			branches: []string{"test", "main", "main/foo"},
		},
		{
			name:  "with ref with nested dir",
			ghURL: "https://github.com/owner/repo/tree/foobranch/my/nested/pkg",
			want:  "https://github.com/owner/repo.git/my/nested/pkg@foobranch",
			errS:  "", branches: []string{"test", "main", "foobranch"},
		},
		{
			name:     "with ref with nested dir ambiguous",
			ghURL:    "https://github.com/owner/repo/tree/foobranch/my/nested/pkg",
			want:     "",
			errS:     "ambiguous repo/dir@version specify '.git' in argument",
			branches: []string{"test", "main", "foobranch/bar"},
		},
		{
			name:     "with ref trailing slash",
			ghURL:    "https://github.com/owner/repo/tree/main/",
			want:     "https://github.com/owner/repo.git@main",
			errS:     "",
			branches: nil,
		},
		{
			name:     "with tree no ref",
			ghURL:    "https://github.com/owner/repo/tree",
			want:     "https://github.com/owner/repo.git/tree",
			errS:     "",
			branches: []string{"test", "tree/"},
		},
		{
			name:     "with tree no ref trailing slash",
			ghURL:    "https://github.com/owner/repo/tree/",
			want:     "https://github.com/owner/repo.git/tree",
			errS:     "",
			branches: []string{"test", "tree/"},
		},
		{
			name:     "with dir no ref",
			ghURL:    "https://github.com/owner/repo/my/nested/pkg",
			want:     "https://github.com/owner/repo.git/my/nested/pkg",
			errS:     "",
			branches: nil,
		},
		{
			name:     "malformed github url domain",
			ghURL:    "https://foo.com/github.com",
			want:     "",
			errS:     "invalid GitHub url",
			branches: nil,
		},
		{
			name:     "malformed github url no repo",
			ghURL:    "https://github.com/owner",
			want:     "",
			errS:     "invalid GitHub pkg url",
			branches: nil,
		},
		{
			name:     "malformed github url no owner no repo",
			ghURL:    "https://github.com/owner",
			want:     "",
			errS:     "invalid GitHub pkg url",
			branches: nil,
		},
		{
			name:     "malformed github url no scheme",
			ghURL:    "github.com/owner",
			want:     "",
			errS:     "invalid GitHub url",
			branches: nil,
		},
		{
			name:     "not github url",
			ghURL:    "https://foo.com/bar",
			want:     "",
			errS:     "invalid GitHub url",
			branches: nil,
		},
		{
			name:     "empty",
			ghURL:    "",
			want:     "",
			errS:     "invalid GitHub url",
			branches: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			ctx := printer.WithContext(context.Background(), printer.New(nil, nil))
			// getBranches returns test branches slice
			getBranches := func(ctx context.Context, repo string) ([]string, error) {
				return tt.branches, nil
			}
			got, err := pkgURLFromGHURL(ctx, tt.ghURL, getBranches)
			if tt.errS != "" {
				r.Equal("", got, "url should be empty")
				r.EqualError(err, fmt.Sprintf("%s: %s", tt.errS, tt.ghURL), "should equal")
			} else {
				r.NoError(err, "should not return error")
				r.Equal(tt.want, got, "url should be equal")
			}
		})
	}
}

func Test_parseURL(t *testing.T) {
	type expected struct {
		repo    string
		dir     string
		version string
	}
	tests := map[string]struct {
		ghURL string
		expected
	}{
		"simple": {
			ghURL:    "https://my.git-server.com/repository.git/directory@main",
			expected: expected{repo: "https://my.git-server.com/repository", dir: "/directory", version: "main"},
		},
		"no ref": {
			ghURL:    "https://github.com/GoogleContainerTools/kpt-functions-catalog.git/examples/apply-replacements-simple",
			expected: expected{repo: "https://github.com/GoogleContainerTools/kpt-functions-catalog", dir: "/examples/apply-replacements-simple", version: ""},
		},
		".git appears in the middle": {
			ghURL:    "https://my-site.com/root.gitops.git/foo@main",
			expected: expected{repo: "https://my-site.com/root.gitops", dir: "/foo", version: "main"},
		},
	}
	for name, test := range tests {
		test := test // capture range variable
		t.Run(name, func(t *testing.T) {
			repo, dir, version, err := URL(test.ghURL)
			assert.NoError(t, err)
			assert.Equal(t, expected{repo: repo, dir: dir, version: version}, test.expected)
		})
	}
}

func Test_GitParseArgs(t *testing.T) {
	tests := map[string]struct {
		ghURL    string
		skip     bool
		expected Target
	}{
		"git@ url with .git suffix": {
			ghURL: "git@github.com:GoogleContainerTools/kpt.git",
			expected: Target{Git: v1.Git{
				Repo:      "git@github.com:GoogleContainerTools/kpt",
				Directory: "/",
				Ref:       "main",
			},
				Destination: "kpt"},
			// in CI we get the error "git@github.com: Permission denied (publickey)" because the ssh key is not set up.
			skip: true,
		},
		"http url with .git suffix": {
			ghURL: "https://github.com/GoogleContainerTools/kpt.git",
			expected: Target{Git: v1.Git{
				Repo:      "https://github.com/GoogleContainerTools/kpt",
				Directory: "/",
				Ref:       "main",
			},
				Destination: "kpt"},
		},
		"starts with github.com": {
			ghURL: "https://github.com/GoogleContainerTools/kpt",
			expected: Target{Git: v1.Git{
				Repo:      "https://github.com/GoogleContainerTools/kpt",
				Directory: "/",
				Ref:       "main",
			},
				Destination: "kpt"},
		},
	}
	for name, test := range tests {
		test := test // capture range variable
		t.Run(name, func(t *testing.T) {
			if test.skip {
				t.SkipNow()
			}
			ctx := printer.WithContext(context.Background(), printer.New(nil, nil))
			actual, err := GitParseArgs(ctx, []string{test.ghURL, ""})
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}
