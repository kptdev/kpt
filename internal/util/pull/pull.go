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

// Package get contains libraries for fetching packages.
package pull

import (
	"context"
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/content/open"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/GoogleContainerTools/kpt/pkg/location"
)

// Command fetches a package from a git repository, copies it to a local
// directory, and expands any remote subpackages.
type Command struct {
	// Contains information about the upstraem package to fetch
	Origin location.Reference

	// Destination is the output directory to clone the package to.  Defaults to the name of the package --
	// either the base repo name, or the base subdirectory name.
	Destination string
}

// Run runs the Command.
func (c Command) Run(ctx context.Context) error {
	const op errors.Op = "pull.Run"
	pr := printer.FromContextOrDie(ctx)
	opts := open.Options(open.WithContext(ctx))

	if err := (&c).DefaultValues(); err != nil {
		return errors.E(op, err)
	}

	if _, err := os.Stat(c.Destination); !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, errors.Exist, types.UniquePath(c.Destination), fmt.Errorf("destination directory already exists"))
	}

	err := os.MkdirAll(c.Destination, 0700)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(c.Destination), err)
	}

	pr.Printf("Pulling origin %s\n", c.Origin.String())

	src, err := open.FileSystem(c.Origin, opts)
	if err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}
	defer src.Close()

	pr.Printf("Pulled digest %s\n", src.ReferenceLock)

	dst, err := open.FileSystem(location.Dir{Directory: c.Destination}, opts)
	if err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}
	defer dst.Close()

	if err := pkgutil.CopyPackage(src.FileSystemPath, dst.FileSystemPath, true, pkg.All); err != nil {
		return errors.E(op, types.UniquePath(c.Destination), err)
	}

	if kf, err := pkg.ReadKptfile(dst.FileSystemPath); err == nil {
		lock, err := kptfileutil.NewUpstreamLockFromReferenceLock(src.ReferenceLock)
		if err != nil {
			return errors.E(op, types.UniquePath(c.Destination), err)
		}

		kf.Origin = &kptfilev1.Origin{
			Type: lock.Type,
			Git:  lock.Git,
			Oci:  lock.Oci,
		}

		err = kptfileutil.WriteFileFS(dst.FileSystemPath, kf)
		if err != nil {
			return errors.E(op, types.UniquePath(c.Destination), err)
		}
	}

	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	const op errors.Op = "pull.DefaultValues"
	if c.Origin == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify git repo or image reference information"))
	}
	if err := c.Origin.Validate(); err != nil {
		return errors.E(op, err)
	}
	if !filepath.IsAbs(c.Destination) {
		return errors.E(op, errors.InvalidParam, fmt.Errorf("destination must be an absolute path"))
	}

	return nil
}
