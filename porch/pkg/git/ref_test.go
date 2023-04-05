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

package git

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

func TestBranchNames(t *testing.T) {
	const main BranchName = "main"

	if got, want := main.RefInRemote(), "refs/heads/main"; string(got) != want {
		t.Errorf("%s in remote repository: got %s, wnat %s", main, got, want)
	}
	if got, want := main.RefInLocal(), "refs/remotes/origin/main"; string(got) != want {
		t.Errorf("%s in local repository: got %s, wnat %s", main, got, want)
	}
}

func TestValidateRefSpecs(t *testing.T) {
	if err := branchRefSpec.Validate(); err != nil {
		t.Errorf("%s validation failed: %v", branchRefSpec, err)
	}
	if err := tagRefSpec.Validate(); err != nil {
		t.Errorf("%s validation failed: %v", tagRefSpec, err)
	}
}

func TestTranslate(t *testing.T) {
	for _, tc := range []struct {
		remote plumbing.ReferenceName
		local  plumbing.ReferenceName
	}{
		{
			remote: "refs/heads/drafts/bucket/v1",
			local:  "refs/remotes/origin/drafts/bucket/v1",
		},
		{
			remote: "refs/tags/bucket/v1",
			local:  "refs/tags/bucket/v1",
		},
		{
			remote: "refs/heads/main",
			local:  "refs/remotes/origin/main",
		},
	} {
		got, err := refInLocalFromRefInRemote(tc.remote)
		if err != nil {
			t.Errorf("refInLocalFromRefInRemote(%s) failed: %v", tc.remote, err)
		}
		if want := tc.local; got != want {
			t.Errorf("refInLocalFromRefInRemote(%s): got %s, want %s", tc.remote, got, want)
		}

		got, err = refInRemoteFromRefInLocal(tc.local)
		if err != nil {
			t.Errorf("refInRemoteFromRefInLocal(%s) failed: %v", tc.local, err)
		}
		if want := tc.remote; got != want {
			t.Errorf("refInRemoteFromRefInLocal(%s): got %s, want %s", tc.local, got, want)
		}
	}
}
