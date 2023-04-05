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

import gogit "github.com/go-git/go-git/v5"

// Repo manages a single git repository
type Repo struct {
	gogit *gogit.Repository

	// Basic auth
	username string
	password string
}

// NewRepo constructs an instance of Repo
func NewRepo(gogit *gogit.Repository, options ...GitRepoOption) (*Repo, error) {
	r := &Repo{
		gogit: gogit,
	}
	for _, option := range options {
		if err := option.apply(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// GitRepoOption is implemented by configuration settings for git repository.
type GitRepoOption interface {
	apply(*Repo) error
}

type optionBasicAuth struct {
	username, password string
}

func (o *optionBasicAuth) apply(s *Repo) error {
	s.username, s.password = o.username, o.password
	return nil
}

func WithBasicAuth(username, password string) GitRepoOption {
	return &optionBasicAuth{
		username: username,
		password: password,
	}
}
