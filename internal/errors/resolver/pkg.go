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
	"os"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&pkgErrorResolver{})
}

// pkgErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the pkg.KptfileError type.
type pkgErrorResolver struct{}

func (*pkgErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var kptfileError *pkg.KptfileError
	if errors.As(err, &kptfileError) {
		path := kptfileError.Path

		return resolveNestedErr(kptfileError, path.String())
	}

	var remoteKptfileError *pkg.RemoteKptfileError
	if errors.As(err, &remoteKptfileError) {
		path := remoteKptfileError.RepoSpec.RepoRef()

		return resolveNestedErr(remoteKptfileError, path)
	}

	var validateError *kptfile.ValidateError
	if errors.As(err, &validateError) {
		return ResolvedResult{
			Message: validateError.Error(),
		}, true
	}

	return ResolvedResult{}, false
}

func resolveNestedErr(err error, path string) (ResolvedResult, bool) {
	if errors.Is(err, os.ErrNotExist) {
		msg := fmt.Sprintf("Error: No Kptfile found at %q.", path)

		return ResolvedResult{
			Message: msg,
		}, true
	}

	var deprecatedv1alpha1KptfileError *pkg.DeprecatedKptfileError
	if errors.As(err, &deprecatedv1alpha1KptfileError) &&
		deprecatedv1alpha1KptfileError.Version == "v1alpha1" {
		msg := fmt.Sprintf("Error: Kptfile at %q has an old version (%q) of the Kptfile schema.\n", path, deprecatedv1alpha1KptfileError.Version)
		msg += "Please update the package to the latest format by following https://kpt.dev/installation/migration."

		return ResolvedResult{
			Message: msg,
		}, true
	}

	var deprecatedv1alpha2KptfileError *pkg.DeprecatedKptfileError
	if errors.As(err, &deprecatedv1alpha2KptfileError) &&
		deprecatedv1alpha2KptfileError.Version == "v1alpha2" {
		msg := fmt.Sprintf("Error: Kptfile at %q has an old version (%q) of the Kptfile schema.\n", path, deprecatedv1alpha2KptfileError.Version)
		msg += "Please run \"kpt fn eval <PKG_PATH> -i gcr.io/kpt-fn/fix:v0.2 --include-meta-resources\" to upgrade the package and retry."

		return ResolvedResult{
			Message: msg,
		}, true
	}

	var unknownKptfileResourceError *pkg.UnknownKptfileResourceError
	if errors.As(err, &unknownKptfileResourceError) {
		msg := fmt.Sprintf("Error: Kptfile at %q has an unknown resource type (%q).", path, unknownKptfileResourceError.GVK.String())
		return ResolvedResult{
			Message: msg,
		}, true
	}

	msg := fmt.Sprintf("Error: Kptfile at %q can't be read.", path)
	if err != nil {
		var kptFileError *pkg.KptfileError
		if errors.As(err, &kptFileError) {
			if kptFileError.Err != nil {
				msg += fmt.Sprintf("\n\nDetails:\n%v", kptFileError.Err)
			}
		} else {
			msg += fmt.Sprintf("\n\nDetails:\n%v", err)
		}
	}

	return ResolvedResult{
		Message: msg,
	}, true
}
