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

package remote

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/gcrane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type ociUpstream struct {
	oci     *kptfilev1.Oci
	ociLock *kptfilev1.OciLock
}

var _ Upstream = &ociUpstream{}

type ociOrigin struct {
	oci *kptfilev1.OciLock
}

var _ Origin = &ociOrigin{}

func NewOciUpstream(oci *kptfilev1.Oci) Upstream {
	return &ociUpstream{
		oci: oci,
	}
}

func NewOciOrigin(oci *kptfilev1.Oci) Origin {
	return &ociOrigin{
		oci: &kptfilev1.OciLock{
			Image: oci.Image,
		},
	}
}

func (u *ociUpstream) String() string {
	return u.oci.Image
}

func (u *ociUpstream) LockedString() string {
	return u.ociLock.Digest
}

func (u *ociOrigin) String() string {
	return u.oci.Image
}

func (u *ociOrigin) LockedString() string {
	return u.oci.Digest
}

func (u *ociUpstream) BuildUpstream() *kptfilev1.Upstream {
	return &kptfilev1.Upstream{
		Type: kptfilev1.OciOrigin,
		Oci:  u.oci,
	}
}

func (u *ociUpstream) BuildUpstreamLock(digest string) *kptfilev1.UpstreamLock {
	u.ociLock.Image = u.oci.Image
	u.ociLock.Directory = u.oci.Directory
	u.ociLock.Digest = digest

	return &kptfilev1.UpstreamLock{
		Type: kptfilev1.OciOrigin,
		Oci:  u.ociLock,
	}
}

func (u *ociOrigin) Build(digest string) *kptfilev1.Origin {
	return &kptfilev1.Origin{
		Type: kptfilev1.OciOrigin,
		Oci: &kptfilev1.OciLock{
			Image:  u.oci.Image,
			Digest: digest,
		},
	}
}

func (u *ociUpstream) Validate() error {
	const op errors.Op = "remote.Validate"
	if u.oci != nil {
		if len(u.oci.Image) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify image"))
		}
	}
	return nil
}

func (u *ociOrigin) Validate() error {
	const op errors.Op = "remote.Validate"
	if u.oci != nil {
		if len(u.oci.Image) == 0 {
			return errors.E(op, errors.MissingParam, fmt.Errorf("must specify image"))
		}
	}
	return nil
}

func (u *ociUpstream) FetchUpstream(ctx context.Context, dest string) (string, string, error) {
	const op errors.Op = "remote.FetchUpstream"
	imageDigest, err := pullAndExtract(u.oci.Image, dest, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return "", "", errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}
	return path.Join(dest, u.oci.Directory), imageDigest.Name(), nil
}

func (u *ociUpstream) FetchUpstreamLock(ctx context.Context, dest string) (string, error) {
	const op errors.Op = "remote.FetchUpstreamLock"
	_, err := pullAndExtract(u.ociLock.Digest, dest, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return "", errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}
	return path.Join(dest, u.ociLock.Directory), nil
}

func (u *ociOrigin) Fetch(ctx context.Context, dest string) (string, string, error) {
	const op errors.Op = "remote.Fetch"
	imageDigest, err := pullAndExtract(u.oci.Image, dest, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return "", "", errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}
	return path.Join(dest, u.oci.Directory), imageDigest.Name(), nil
}

func (u *ociUpstream) CloneUpstream(ctx context.Context, dest string) error {
	const op errors.Op = "remote.FetchUpstreamClone"
	// pr := printer.FromContextOrDie(ctx)

	// We need to create a temp directory where we can copy the content of the repo.
	// During update, we need to checkout multiple versions of the same repo, so
	// we can't do merges directly from the cache.
	dir, err := ioutil.TempDir("", "kpt-get-")
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error creating temp directory: %w", err))
	}
	defer os.RemoveAll(dir)

	imageDigest, err := pullAndExtract(u.oci.Image, dir, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}

	sourcePath := path.Join(dir, u.oci.Directory)
	if err := pkgutil.CopyPackage(types.DiskPath(sourcePath), types.DiskPath(dest), true, pkg.All); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateKptfileWithoutOrigin(types.DiskPath(dest), types.DiskPath(sourcePath), false); err != nil {
		return errors.E(op, types.UniquePath(dest), err)
	}

	if err := kptfileutil.UpdateUpstreamLock(types.DiskPath(dest), u.BuildUpstreamLock(imageDigest.String())); err != nil {
		return errors.E(op, errors.OCI, types.UniquePath(dest), err)
	}

	return nil
}

func (u *ociOrigin) Push(ctx context.Context, source string, kptfile *kptfilev1.KptFile) (digest string, err error) {
	const op errors.Op = "remote.Push"

	imageDigest, err := archiveAndPush(u.oci.Image, source, kptfile, remote.WithContext(ctx), remote.WithAuthFromKeychain(gcrane.Keychain))
	if err != nil {
		return "", errors.E(op, errors.OCI, types.UniquePath(source), err)
	}

	return imageDigest.String(), nil
}

func (u *ociUpstream) Ref() (string, error) {
	const op errors.Op = "remote.Ref"
	r, err := name.ParseReference(u.oci.Image)
	if err != nil {
		return "", errors.E(op, errors.Internal, fmt.Errorf("error parsing reference: %s %w", u.oci.Image, err))
	}
	return r.Identifier(), nil
}

func (u *ociUpstream) SetRef(ref string) error {
	const op errors.Op = "remote.SetRef"
	r, err := name.ParseReference(u.oci.Image)
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error parsing reference: %s %w", u.oci.Image, err))
	}

	if len(strings.SplitN(ref, "sha256:", 2)[0]) == 0 {
		u.oci.Image = r.Context().Digest(ref).Name()
	} else {
		u.oci.Image = r.Context().Tag(ref).Name()
	}

	return nil
}

func (u *ociOrigin) Ref() (string, error) {
	const op errors.Op = "remote.Ref"
	r, err := name.ParseReference(u.oci.Image)
	if err != nil {
		return "", errors.E(op, errors.Internal, fmt.Errorf("error parsing reference: %s %w", u.oci.Image, err))
	}
	return r.Identifier(), nil
}

func (u *ociOrigin) SetRef(ref string) error {
	const op errors.Op = "remote.SetRef"
	r, err := name.ParseReference(u.oci.Image)
	if err != nil {
		return errors.E(op, errors.Internal, fmt.Errorf("error parsing reference: %s %w", u.oci.Image, err))
	}

	if len(strings.SplitN(ref, "sha256:", 2)[0]) == 0 {
		u.oci.Image = r.Context().Digest(ref).Name()
	} else {
		u.oci.Image = r.Context().Tag(ref).Name()
	}

	return nil
}

// shouldUpdateSubPkgRef checks if subpkg ref should be updated.
// This is true if pkg has the same upstream repo, upstream directory is within or equal to root pkg directory and original root pkg ref matches the subpkg ref.
func (u *ociUpstream) ShouldUpdateSubPkgRef(rootUpstream Upstream, originalRootKfRef string) bool {
	root, ok := rootUpstream.(*ociUpstream)
	if !ok {
		return false
	}
	subName, err := name.ParseReference(u.oci.Image)
	if err != nil {
		return false
	}
	rootName, err := name.ParseReference(root.oci.Image)
	if err != nil {
		return false
	}
	return subName.Context().String() == rootName.Context().String() &&
		subName.Identifier() == originalRootKfRef
}

// pullAndExtract uses current credentials (gcloud auth) to pull and
// extract (untar) image files to target directory. The desired version or digest must
// be in the imageName, and the resolved image sha256 digest is returned.
func pullAndExtract(imageName string, dir string, options ...remote.Option) (name.Reference, error) {
	const op errors.Op = "remote.pullAndExtract"

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

// archiveAndPush uses current credentials (gcloud auth) to tar and
// extract (untar) image files to target directory. The desired version or digest must
// be in the imageName, and the resolved image sha256 digest is returned.
func archiveAndPush(imageName string, dir string, kptfile *kptfilev1.KptFile, options ...remote.Option) (name.Reference, error) {
	const op errors.Op = "remote.archiveAndPush"

	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %v", imageName, err)
	}

	// Make new layer
	tarFile, err := ioutil.TempFile("", "tar")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tarFile.Name())

	if err := func() error {
		defer tarFile.Close()

		gw := gzip.NewWriter(tarFile)
		defer gw.Close()

		tw := tar.NewWriter(gw)
		defer tw.Close()

		if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relative, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if info.IsDir() && relative == "." {
				return nil
			}

			// TODO(oci-support) if info is symlink also read link target
			link := ""

			// generate tar header
			header, err := tar.FileInfoHeader(info, link)
			if err != nil {
				return err
			}

			// must provide real name
			// (see https://golang.org/src/archive/tar/common.go?#L626)
			header.Name = filepath.ToSlash(relative)

			var buf *bytes.Buffer
			if strings.EqualFold(header.Name, "Kptfile") {
				buf = &bytes.Buffer{}
				if err := kptfileutil.Write(buf, kptfile); err != nil {
					return err
				}
				header.Size = int64(buf.Len())
			}

			// write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			// if not a dir, write file content
			if !info.IsDir() {
				data, err := os.Open(path)
				if err != nil {
					return err
				}
				if buf != nil {
					if _, err := io.Copy(tw, buf); err != nil {
						return err
					}
				} else {
					if _, err := io.Copy(tw, data); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}

		return nil
	}(); err != nil {
		return nil, err
	}

	// Append new layer
	newLayers := []string{tarFile.Name()}
	img, err := crane.Append(empty.Image, newLayers...)
	if err != nil {
		return nil, fmt.Errorf("appending %v: %v", newLayers, err)
	}

	if err := remote.Write(ref, img, options...); err != nil {
		return nil, fmt.Errorf("pushing image %s: %v", ref, err)
	}

	// Determine the digest of the image that was pushed
	imageDigestHash, err := img.Digest()
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("error calculating image digest: %w", err))
	}
	imageDigest := ref.Context().Digest("sha256:" + imageDigestHash.Hex)

	// Return the image with digest when successful, needed for upstreamLock
	return imageDigest, nil
}
