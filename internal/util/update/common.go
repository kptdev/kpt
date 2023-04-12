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

package update

import (
	"reflect"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// PkgHasUpdatedUpstream checks if the the local package has different
// upstream information than origin.
func PkgHasUpdatedUpstream(local, origin string) (bool, error) {
	const op errors.Op = "update.PkgHasUpdatedUpstream"
	originKf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, origin)
	if err != nil {
		return false, errors.E(op, types.UniquePath(local), err)
	}

	localKf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, local)
	if err != nil {
		return false, errors.E(op, types.UniquePath(local), err)
	}

	// If the upstream information in local has changed from origin, it
	// means the user had updated the package independently and we don't
	// want to override it.
	if !reflect.DeepEqual(localKf.Upstream.Git, originKf.Upstream.Git) {
		return true, nil
	}
	return false, nil
}
