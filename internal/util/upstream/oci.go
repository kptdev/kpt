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

package upstream

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type ociUpstream struct {
	image string
}

var _ Fetcher = &ociUpstream{}

func NewOciUpstream(oci *v1.Oci) Fetcher {
	return &ociUpstream{
		image: oci.Image,
	}
}

func (u *ociUpstream) String() string {
	return u.image
}

func (u *ociUpstream) ApplyUpstream(kf *v1.KptFile) {

	kf.Upstream = &v1.Upstream{
		Type: v1.OciOrigin,
		Oci: &v1.Oci{
			Image: u.image,
		},
	}
}

func (u *ociUpstream) Validate() error {
	const op errors.Op = "upstream.Validate"
	if len(u.image) == 0 {
		return errors.E(op, errors.MissingParam, fmt.Errorf("must specify image"))
	}
	return nil
}

func (u *ociUpstream) FetchUpstream(ctx context.Context, dest string) error {
	const op errors.Op = "upstream.FetchUpstream"
	// pr := printer.FromContextOrDie(ctx)

	// We need to create a temp directory where we can copy the content of the repo.
	// During update, we need to checkout multiple versions of the same repo, so
	// we can't do merges directly from the cache.
	dir, err := ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
	}
	defer os.RemoveAll(dir)

	imageDigest, err := pullAndExtract(u.image, dir, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}

	sourcePath := dir
	if err := pkgutil.CopyPackage(sourcePath, dest, true, pkg.All); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateKptfileWithoutOrigin(dest, sourcePath, false); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateUpstreamLockFromOCI(dest, imageDigest); err != nil {
		return errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}

	return nil
}

// pullAndExtract uses current credentials (gcloud auth) to pull and
// extract (untar) image files to target directory. The desired version or digest must
// be in the imageName, and the resolved image sha256 digest is returned.
func pullAndExtract(imageName string, dir string, options ...remote.Option) (name.Reference, error) {
	const op errors.Op = "upstream.pullAndExtract"

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %v", imageName, err)
	}

	// Pull image from source using provided options for auth credentials
	image, err := remote.Image(ref, options...)
	if err != nil {
		return nil, fmt.Errorf("pulling image %s: %v", imageName, err)
	}

	// Stream image files as if single tar (merged layers)
	ioReader := mutate.Extract(image)
	defer ioReader.Close()

	// Write contents to target dir
	// TODO look for a more robust example of an untar loop
	tarReader := tar.NewReader(ioReader)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		path := filepath.Join(dir, hdr.Name)
		switch {
		case hdr.FileInfo().IsDir():
			if err := os.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return nil, err
			}
		case hdr.Linkname != "":
			if err := os.Symlink(hdr.Linkname, path); err != nil {
				// just warn for now
				fmt.Fprintln(os.Stderr, err)
				// return err
			}
		default:
			file, err := os.OpenFile(path,
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				os.FileMode(hdr.Mode),
			)
			if err != nil {
				return nil, err
			}
			defer file.Close()

			_, err = io.Copy(file, tarReader)
			if err != nil {
				return nil, err
			}
		}
	}

	// Determine the digest of the image that was extracted
	imageDigestHash, err := image.Digest()
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("error calculating image digest: %w", err))
	}
	imageDigest := ref.Context().Digest("sha256:" + imageDigestHash.Hex)

	// Return the image with digest when successful, needed for upstreamLock
	return imageDigest, nil
}
