// Copyright 2021 Google LLC
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_pkgURLFromGHURL(t *testing.T) {
	tests := []struct {
		name  string
		ghURL string
		want  string
		errS  string
	}{
		{"simple", "https://github.com/owner/repo", "https://github.com/owner/repo.git", ""},
		{"with trailing slash", "https://github.com/owner/repo/", "https://github.com/owner/repo.git", ""},
		{"with ref", "https://github.com/owner/repo/tree/main", "https://github.com/owner/repo.git@main", ""},
		{"with ref with nested dir", "https://github.com/owner/repo/tree/foobranch/my/nested/pkg", "https://github.com/owner/repo.git/my/nested/pkg@foobranch", ""},
		{"with ref trailing slash", "https://github.com/owner/repo/tree/main/", "https://github.com/owner/repo.git@main", ""},
		{"with tree no ref", "https://github.com/owner/repo/tree", "https://github.com/owner/repo.git/tree", ""},
		{"with tree no ref trailing slash", "https://github.com/owner/repo/tree/", "https://github.com/owner/repo.git/tree", ""},
		{"with dir no ref", "https://github.com/owner/repo/my/nested/pkg", "https://github.com/owner/repo.git/my/nested/pkg", ""},
		{"malformed github url domain", "https://foo.com/github.com", "", "invalid GitHub url"},
		{"malformed github url no repo", "https://github.com/owner", "", "invalid GitHub pkg url"},
		{"malformed github url no owner no repo", "https://github.com/owner", "", "invalid GitHub pkg url"},
		{"malformed github url no scheme", "github.com/owner", "", "invalid GitHub url"},
		{"not github url", "https://foo.com/bar", "", "invalid GitHub url"},
		{"empty", "", "", "invalid GitHub url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := require.New(t)
			got, err := pkgURLFromGHURL(tt.ghURL)
			if tt.errS != "" {
				r.Equal("", got, "url should be empty")
				r.EqualError(err, fmt.Sprintf("%s: %s", tt.errS, tt.ghURL), "should equal")
			} else {
				r.Equal(tt.want, got, "url should be equal")
				r.NoError(err, "should not return error")
			}
		})
	}
}
