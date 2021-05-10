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

package resolver

import (
	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&updateErrorResolver{})
}

var (
	//nolint:lll
	pkgNotGitRepo = `
Package {{ printf "%q" .repo }} is not within a git repository. Please initialize a repository using 'git init' and then commit the changes using 'git commit -m "<commit message>"'.
`

	pkgRepoDirty = `
Package {{ printf "%q" .repo }} contains uncommitted changes. Please commit the changes using 'git commit -m "<commit message>"'.
`
)

// updateErrorResolver is an implementation of the ErrorResolver interface
// to resolve update errors.
type updateErrorResolver struct{}

func (*updateErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var msg string

	var pkgNotGitRepoError *update.PkgNotGitRepoError
	if errors.As(err, &pkgNotGitRepoError) {
		msg = ExecuteTemplate(pkgNotGitRepo, map[string]interface{}{
			"repo": pkgNotGitRepoError.Path,
		})
	}

	var pkgRepoDirtyError *update.PkgRepoDirtyError
	if errors.As(err, &pkgRepoDirtyError) {
		msg = ExecuteTemplate(pkgRepoDirty, map[string]interface{}{
			"repo": pkgRepoDirtyError.Path,
		})
	}

	if msg != "" {
		return ResolvedResult{
			Message:  msg,
			ExitCode: 1,
		}, true
	}
	return ResolvedResult{}, false
}
