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
	"os"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&pkgErrorResolver{})
}

const (
	noKptfileMsg = `
Error: No Kptfile found at {{ printf "%q" .path }}.
`

	kptfileReadErrMsg = `
Error: Kptfile at {{ printf "%q" .path }} can't be read.

{{- template "NestedErrDetails" . }}
`
)

// pkgErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the pkg.KptfileError type.
type pkgErrorResolver struct{}

func (*pkgErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var kptfileError *pkg.KptfileError
	if errors.As(err, &kptfileError) {
		path := kptfileError.Path
		tmplArgs := map[string]interface{}{
			"path": path,
			"err":  kptfileError,
		}

		if errors.Is(kptfileError, os.ErrNotExist) {
			return ResolvedResult{
				Message: ExecuteTemplate(noKptfileMsg, tmplArgs),
			}, true
		}

		return ResolvedResult{
			Message: ExecuteTemplate(kptfileReadErrMsg, tmplArgs),
		}, true
	}

	return ResolvedResult{}, false
}
