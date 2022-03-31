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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// This file contains helpers for interacting with gogit.

func InitEmptyRepository(path string) (*git.Repository, error) {
	isBare := true // Porch only uses bare repositories
	repo, err := git.PlainInit(path, isBare)
	if err != nil {
		return nil, err
	}
	if err := RemoveDefaultBranches(repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func RemoveDefaultBranches(repo *git.Repository) error {
	// Adjust default references
	if err := repo.Storer.RemoveReference(plumbing.Master); err != nil {
		return err
	}
	// gogit points HEAD at a wrong branch; delete it too
	if err := repo.Storer.RemoveReference(plumbing.HEAD); err != nil {
		return err
	}
	return nil
}
