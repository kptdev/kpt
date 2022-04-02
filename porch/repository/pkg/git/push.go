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

package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

type pushHelper struct {
	repo     *git.Repository
	pushRefs map[plumbing.ReferenceName]plumbing.Hash
	cond     map[plumbing.ReferenceName]plumbing.Hash
}

func newPushHelper(repo *git.Repository) *pushHelper {
	return &pushHelper{
		repo:     repo,
		pushRefs: map[plumbing.ReferenceName]plumbing.Hash{},
		cond:     map[plumbing.ReferenceName]plumbing.Hash{},
	}
}

func (h *pushHelper) Push(hash plumbing.Hash, to plumbing.ReferenceName) {
	h.pushRefs[to] = hash
}

func (h *pushHelper) Delete(ref *plumbing.Reference) {
	if ref != nil {
		h.Push(plumbing.ZeroHash, ref.Name())
		h.RequireRef(ref)
	}
}

func (h *pushHelper) RequireRef(ref *plumbing.Reference) {
	if ref != nil {
		h.cond[ref.Name()] = ref.Hash()
	}
}

func (h *pushHelper) CreateSpecs() (push []config.RefSpec, require []config.RefSpec, err error) {
	for local, hash := range h.pushRefs {
		remote, err := refInRemoteFromRefInLocal(local)
		if err != nil {
			return nil, nil, err
		}

		if !hash.IsZero() {
			push = append(push, config.RefSpec(fmt.Sprintf("%s:%s", hash, remote)))
		} else {
			push = append(push, config.RefSpec(fmt.Sprintf(":%s", remote)))
		}
	}

	for local, hash := range h.cond {
		remote, err := refInRemoteFromRefInLocal(local)
		if err != nil {
			return nil, nil, err
		}
		if !hash.IsZero() {
			require = append(require, config.RefSpec(fmt.Sprintf("%s:%s", hash, remote)))
		}
	}

	return push, require, nil
}
