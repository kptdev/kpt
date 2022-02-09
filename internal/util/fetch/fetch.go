// Copyright 2019 Google LLC
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

package fetch

import (
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

// Command takes the upstream information in the Kptfile at the path for the
// provided package, and fetches the package referenced if it isn't already
// there.
type Command struct {
	Pkg *pkg.Pkg
}

// Run runs the Command.
func (c Command) Run(ctx context.Context) error {
	const op errors.Op = "fetch.Run"
	pr := printer.FromContextOrDie(ctx)

	kf, err := c.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("no Kptfile found"))
	}

	if err := c.validate(kf); err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	// upstream source location
	srcRef, err := kptfileutil.NewReferenceFromUpstream(kf)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	// open upstream as filesystem
	src, err := open.FileSystem(srcRef, open.WithContext(ctx))
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	defer src.Close()

	// destination package location
	dstRef := location.Dir{
		Directory: c.Pkg.UniquePath.String(),
	}

	// open destination as filesystem
	dst, err := open.FileSystem(dstRef)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}
	defer dst.Close()

	pr.Printf("Adding package %q.\n", src.Reference)

	// copy package from source to destination
	if err := pkgutil.CopyPackage(src.FileSystemPath, dst.FileSystemPath, true, pkg.All); err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	if err := kptfileutil.UpdateKptfileWithoutOrigin(dst.FileSystemPath, src.FileSystemPath, false); err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	if err := kptfileutil.UpdateUpstreamLocations(dst.FileSystemPath, src.Location); err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	result, err := content.Commit(dst)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	pr.Printf("Updated %q.\n", result)

	return nil
}

// validate makes sure the Kptfile has the necessary information to fetch
// the package.
func (c Command) validate(kf *kptfilev1.KptFile) error {
	const op errors.Op = "validate"
	if kf.Upstream == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile doesn't contain upstream information"))
	}
	switch kf.Upstream.Type {
	case kptfilev1.GitOrigin:
		if kf.Upstream.Git == nil {
			return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream doesn't have git information"))
		}

		g := kf.Upstream.Git
		if len(g.Repo) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify repo"))
		}
		if len(g.Ref) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify ref"))
		}
		if len(g.Directory) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify directory"))
		}

	case kptfilev1.OciOrigin:
		if kf.Upstream.Oci == nil {
			return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream doesn't have oci information"))
		}
		if len(kf.Upstream.Oci.Image) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify image"))
		}

	default:
		return errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream type must be one of: %s,%s", kptfilev1.GitOrigin, kptfilev1.OciOrigin))
	}

	return nil
}
