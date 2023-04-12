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

package resolver

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&updateErrorResolver{})
}

// updateErrorResolver is an implementation of the ErrorResolver interface
// to resolve update errors.
type updateErrorResolver struct{}

func (*updateErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var msg string

	var pkgNotGitRepoError *update.PkgNotGitRepoError
	if errors.As(err, &pkgNotGitRepoError) {
		//nolint:lll
		msg = fmt.Sprintf("Package %q is not within a git repository.", pkgNotGitRepoError.Path)
		msg += " Please initialize a repository using 'git init' and then commit the changes using 'git commit -m \"<commit message>\"'."
	}

	var pkgRepoDirtyError *update.PkgRepoDirtyError
	if errors.As(err, &pkgRepoDirtyError) {
		msg = fmt.Sprintf("Package %q contains uncommitted changes.", pkgRepoDirtyError.Path)
		msg += " Please commit the changes using 'git commit -m \"<commit message>\"'."
	}

	if msg != "" {
		return ResolvedResult{
			Message: msg,
		}, true
	}
	return ResolvedResult{}, false
}
