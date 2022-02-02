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

package location

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/google/go-containerregistry/pkg/name"
)

type Oci struct {
	Image     name.Reference
	Directory string
}

var _ Reference = Oci{}
var _ DirectoryNameDefaulter = Oci{}

type OciLock struct {
	Oci
	Digest name.Reference
}

var _ Reference = OciLock{}
var _ ReferenceLock = OciLock{}

func NewOci(location string, opts ...Option) (Oci, error) {

	if s, ok := startsWith(location, "oci://"); ok {
		ref, err := name.ParseReference(s)
		if err != nil {
			return Oci{}, err
		}
		if parts := strings.SplitN(ref.Context().Name(), "//", 2); len(parts) == 2 {
			repo, err := name.NewRepository(parts[0])
			if err != nil {
				return Oci{}, err
			}

			dir := filepath.Clean(parts[1])
			if filepath.IsAbs(dir) {
				dir, err = filepath.Rel("/", dir)
				if err != nil {
					return Oci{}, err
				}
			}

			switch ref := ref.(type) {
			case name.Tag:
				return Oci{
					Image:     repo.Tag(ref.TagStr()),
					Directory: dir,
				}, nil
			case name.Digest:
				return Oci{
					Image:     repo.Digest(ref.DigestStr()),
					Directory: dir,
				}, nil
			}
		}
		return Oci{
			Image:     ref,
			Directory: ".",
		}, nil
	}

	return Oci{}, fmt.Errorf("invalid format")
}

func parseOci(value string) (Reference, error) {
	if _, ok := startsWith(value, "oci://"); ok {
		return NewOci(value)
	}
	return nil, nil
}

// Type implements location.Reference
func (ref Oci) String() string {
	return fmt.Sprintf("type:oci image:%q directory:%q", ref.Image, ref.Directory)
}

// Type implements location.ReferenceLock
func (ref OciLock) String() string {
	return fmt.Sprintf("%v digest:%q", ref.Oci, ref.Digest)
}

// Validate implements location.Reference
func (ref Oci) Validate() error {
	const op errors.Op = "oci.Validate"
	if ref.Image == nil {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify image"))
	}
	return nil
}

// Type implements location.Reference
func (ref Oci) Type() string {
	return "oci"
}

// GetDefaultDirectoryName is called from location.DefaultDirectoryName
func (ref Oci) GetDefaultDirectoryName() (string, bool) {
	return path.Base(path.Join(path.Clean(ref.Image.Context().Name()), path.Clean(ref.Directory))), true
}

// SetIdentifier is called from mutate.Identifier
func (ref Oci) SetIdentifier(identifier string) (Reference, error) {
	return Oci{
		Image:     ref.Image.Context().Tag(identifier),
		Directory: ref.Directory,
	}, nil
}

// SetLock is called from mutate.Lock
func (ref Oci) SetLock(lock string) (ReferenceLock, error) {
	return OciLock{
		Oci:    ref,
		Digest: ref.Image.Context().Digest(lock),
	}, nil
}
