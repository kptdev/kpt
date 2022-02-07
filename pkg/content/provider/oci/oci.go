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

package oci

import (
	"archive/tar"
	"io"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	locationmutate "github.com/GoogleContainerTools/kpt/pkg/location/mutate"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type ociProvider struct {
	image v1.Image
	fsys  filesys.FileSystem
}

var _ extensions.FileSystemProvider = &ociProvider{}

func Open(ref location.Oci, options ...remote.Option) (content.Content, location.ReferenceLock, error) {
	return open(ref.Image, ref, options...)
}

func OpenLock(ref location.OciLock, options ...remote.Option) (content.Content, location.ReferenceLock, error) {
	return open(ref.Digest, ref, options...)
}

func open(name name.Reference, ref location.Reference, options ...remote.Option) (content.Content, location.ReferenceLock, error) {
	image, err := remote.Image(name, options...)
	if err != nil {
		return nil, nil, err
	}

	// Determine the digest of the image that was extracted
	imageDigestHash, err := image.Digest()
	if err != nil {
		return nil, nil, err
		// return nil, errors.E(op, fmt.Errorf("error calculating image digest: %w", err))
	}

	lock, err := locationmutate.Lock(ref, "sha256:"+imageDigestHash.Hex)
	if err != nil {
		return nil, nil, err
	}

	return &ociProvider{
		image: image,
	}, lock, nil
}

func (p *ociProvider) Close() error {
	return nil
}

func (p *ociProvider) ProvideFileSystem() (filesys.FileSystem, string, error) {
	if p.fsys != nil {
		return p.fsys, "/", nil
	}

	fsys := filesys.MakeFsInMemory()

	if err := pullAndExtract(p.image, fsys, "/"); err != nil {
		return nil, "", err
	}

	p.fsys = fsys
	return fsys, "/", nil
}

// pullAndExtract uses current credentials (gcloud auth) to pull and
// extract (untar) image files to target directory. The desired version or digest must
// be in the imageName, and the resolved image sha256 digest is returned.
func pullAndExtract(image v1.Image, fsys filesys.FileSystem, dir string) error {
	// const op errors.Op = "oci.pullAndExtract"

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
			return err
		}
		path := filepath.Join(dir, hdr.Name)
		switch {
		case hdr.FileInfo().IsDir():
			if err := fsys.MkdirAll(path); err != nil {
				return err
			}
		default:
			file, err := fsys.Create(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(file, tarReader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
