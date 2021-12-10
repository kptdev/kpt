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
package push

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/printer"
	"github.com/GoogleContainerTools/kpt/internal/util/remote"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/Masterminds/semver"
)

// Command fetches a package from a git repository, copies it to a local
// directory, and expands any remote subpackages.
type Command struct {
	// Pkg captures information about the package that should be push.
	Pkg *pkg.Pkg

	// Ref is the version to push to origin
	Ref string

	// Contains information about the package origin
	Origin remote.Origin

	// Increment determines is the version portion of the reference should be increased
	Increment bool

	// Origin assigns remote location for push. Ref and Increment will alter parts of this value.
	Path string
}

// Run runs the Command.
func (c Command) Run(ctx context.Context) error {
	const op errors.Op = "push.Run"
	pr := printer.FromContextOrDie(ctx)

	if err := (&c).DefaultValues(); err != nil {
		return errors.E(op, err)
	}

	path := c.Pkg.UniquePath.String()
	kf, err := c.Pkg.Kptfile()
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	if c.Origin == nil {
		c.Origin, err = remote.NewOrigin(kf)
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("package must have an origin reference: %v", err))
		}
	}

	if c.Ref != "" {
		if err := c.Origin.SetRef(c.Ref); err != nil {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("error updating ref: %v", err))
		}
	}

	if c.Increment {
		// TODO(dejardin) move this logic into a util with test coverage
		ref, err := c.Origin.Ref()
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("missing origin version information: %v", err))
		}

		prefix := ""
		if ref != "" && ref[:1] == "v" {
			prefix = "v"
		}

		dotParts := len(strings.SplitN(ref, ".", 3))
		if dotParts > 3 {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("origin version '%s' has more than three dotted parts", ref))
		}

		v, err := semver.NewVersion(ref)
		if err != nil {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("unable to increment '%s': %v", ref, err))
		}

		var buf bytes.Buffer
		switch dotParts {
		case 1:
			fmt.Fprintf(&buf, "%s%d", prefix, v.Major()+1)
		case 2:
			fmt.Fprintf(&buf, "%s%d.%d", prefix, v.Major(), v.Minor()+1)
		case 3:
			fmt.Fprintf(&buf, "%s%d.%d.%d", prefix, v.Major(), v.Minor(), v.Patch()+1)
		}
		if v.Prerelease() != "" {
			fmt.Fprintf(&buf, "-%s", v.Prerelease())
		}
		if v.Metadata() != "" {
			fmt.Fprintf(&buf, "+%s", v.Metadata())
		}

		pr.Printf("Incrementing %s to %s\n", ref, buf.String())

		if err := c.Origin.SetRef(buf.String()); err != nil {
			return errors.E(op, c.Pkg.UniquePath, fmt.Errorf("error updating ref: %v", err))
		}
	}

	// the kptfile pushed in the package does not have origin data
	// this is because the digest will be incorrect. Also, if it is
	// pulled from a different location or via different branch, the
	// correct origin will be added as part of the pull operation.
	kf.Origin = nil

	pr.Printf("Pushing origin %s\n", c.Origin.String())

	digest, err := c.Origin.Push(ctx, path, kf)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	pr.Printf("Pushed digest %s\n", digest)

	kf.Origin = c.Origin.Build(digest)
	err = kptfileutil.WriteFile(path, kf)
	if err != nil {
		return errors.E(op, c.Pkg.UniquePath, err)
	}

	return nil
}

// DefaultValues sets values to the default values if they were unspecified
func (c *Command) DefaultValues() error {
	// const op errors.Op = "pull.DefaultValues"
	// if c.Origin == nil {
	// 	return errors.E(op, errors.MissingParam, fmt.Errorf("must specify git repo or image reference information"))
	// }
	// if err := c.Origin.Validate(); err != nil {
	// 	return errors.E(op, err)
	// }

	return nil
}
