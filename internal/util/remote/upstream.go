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
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

type Fetcher interface {
	fmt.Stringer
	LockedString() string
	OriginString() string

	Validate() error
	BuildUpstream() *kptfilev1.Upstream
	BuildUpstreamLock(digest string) *kptfilev1.UpstreamLock
	BuildOrigin(digest string) *kptfilev1.Origin

	FetchUpstream(ctx context.Context, dest string) (absPath string, digest string, err error)
	FetchUpstreamLock(ctx context.Context, dest string) (absPath string, err error)
	FetchOrigin(ctx context.Context, dest string) (absPath string, digest string, err error)
	PushOrigin(ctx context.Context, dest string, kptfile *kptfilev1.KptFile) (digest string, err error)

	CloneUpstream(ctx context.Context, dest string) error

	Ref() (string, error)
	SetRef(ref string) error
	ShouldUpdateSubPkgRef(rootUpstream Fetcher, originalRootRef string) bool

	OriginRef() (string, error)
	SetOriginRef(ref string) error
}

func NewUpstream(kf *kptfilev1.KptFile) (Fetcher, error) {
	const op errors.Op = "remote.NewUpstream"
	if kf != nil && kf.Upstream != nil {
		switch kf.Upstream.Type {
		case kptfilev1.GitOrigin:
			if kf.Upstream.Git == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream must have git information"))
			}
			u := &gitUpstream{
				git:     kf.Upstream.Git,
				gitLock: &kptfilev1.GitLock{},
			}
			if kf.UpstreamLock != nil && kf.UpstreamLock.Git != nil {
				u.gitLock = kf.UpstreamLock.Git
			}
			return u, nil
		case kptfilev1.OciOrigin:
			if kf.Upstream.Oci == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream must have oci information"))
			}
			u := &ociUpstream{
				oci:     kf.Upstream.Oci,
				ociLock: &kptfilev1.OciLock{},
			}
			if kf.UpstreamLock != nil && kf.UpstreamLock.Oci != nil {
				u.ociLock = kf.UpstreamLock.Oci
			}
			return u, nil
		}
	}
	return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream type must be one of: %s,%s", kptfilev1.GitOrigin, kptfilev1.OciOrigin))
}

func NewOrigin(kf *kptfilev1.KptFile) (Fetcher, error) {
	const op errors.Op = "remote.NewOrigin"
	if kf != nil && kf.Origin != nil {
		switch kf.Origin.Type {
		case kptfilev1.GitOrigin:
			if kf.Upstream.Git == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin must have git information"))
			}
			u := &gitUpstream{
				origin: kf.Origin.Git,
			}
			return u, nil
		case kptfilev1.OciOrigin:
			if kf.Origin.Oci == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin must have oci information"))
			}
			u := &ociUpstream{
				origin: kf.Origin.Oci,
			}
			return u, nil
		}
	}
	return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin type must be one of: %s,%s", kptfilev1.GitOrigin, kptfilev1.OciOrigin))
}

func ShouldUpdateSubPkgRef(subUpstream Fetcher, rootUpstream Fetcher, originalRootRef string) bool {
	return subUpstream.ShouldUpdateSubPkgRef(rootUpstream, originalRootRef)
}
