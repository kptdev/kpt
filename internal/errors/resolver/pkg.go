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
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&pkgErrorResolver{})
}

const (
	noKptfileMsg = `
Error: No Kptfile found at {{ printf "%q" .path }}.
`

	//nolint:lll
	deprecatedv1Alpha1KptfileMsg = `
Error: Kptfile at {{ printf "%q" .path }} has an old version ({{ printf "%q" .version }}) of the Kptfile schema.
Please update the package to the latest format by following https://kpt.dev/installation/migration.
`

	deprecatedv1Alpha2KptfileMsg = `
Error: Kptfile at {{ printf "%q" .path }} has an old version ({{ printf "%q" .version }}) of the Kptfile schema.
Please run "kpt fn eval <PKG_PATH> -i gcr.io/kpt-fn/fix:v0.2 --include-meta-resources" to upgrade the package and retry.
`

	unknownKptfileResourceMsg = `
Error: Kptfile at {{ printf "%q" .path }} has an unknown resource type ({{ printf "%q" .gvk.String }}).
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

		return resolveNestedErr(kptfileError, tmplArgs)
	}

	var remoteKptfileError *pkg.RemoteKptfileError
	if errors.As(err, &remoteKptfileError) {
		path := remoteKptfileError.RepoSpec.RepoRef()
		tmplArgs := map[string]interface{}{
			"path": path,
			"err":  kptfileError,
		}
		return resolveNestedErr(remoteKptfileError, tmplArgs)
	}

	var validateError *kptfile.ValidateError
	if errors.As(err, &validateError) {
		return ResolvedResult{
			Message: validateError.Error(),
		}, true
	}

	return ResolvedResult{}, false
}

func resolveNestedErr(err error, tmplArgs map[string]interface{}) (ResolvedResult, bool) {
	if errors.Is(err, os.ErrNotExist) {
		return ResolvedResult{
			Message: ExecuteTemplate(noKptfileMsg, tmplArgs),
		}, true
	}

	var deprecatedv1alpha1KptfileError *pkg.DeprecatedKptfileError
	if errors.As(err, &deprecatedv1alpha1KptfileError) &&
		deprecatedv1alpha1KptfileError.Version == pkg.DeprecatedKptfileVersions[0] {
		tmplArgs["version"] = deprecatedv1alpha1KptfileError.Version
		errMsg := deprecatedv1Alpha1KptfileMsg
		return ResolvedResult{
			Message: ExecuteTemplate(errMsg, tmplArgs),
		}, true
	}

	var deprecatedv1alpha2KptfileError *pkg.DeprecatedKptfileError
	if errors.As(err, &deprecatedv1alpha2KptfileError) &&
		deprecatedv1alpha1KptfileError.Version == pkg.DeprecatedKptfileVersions[1] {
		tmplArgs["version"] = deprecatedv1alpha2KptfileError.Version
		errMsg := deprecatedv1Alpha2KptfileMsg
		return ResolvedResult{
			Message: ExecuteTemplate(errMsg, tmplArgs),
		}, true
	}

	var unknownKptfileResourceError *pkg.UnknownKptfileResourceError
	if errors.As(err, &unknownKptfileResourceError) {
		tmplArgs["gvk"] = unknownKptfileResourceError.GVK
		return ResolvedResult{
			Message: ExecuteTemplate(unknownKptfileResourceMsg, tmplArgs),
		}, true
	}

	return ResolvedResult{
		Message: ExecuteTemplate(kptfileReadErrMsg, tmplArgs),
	}, true
}
