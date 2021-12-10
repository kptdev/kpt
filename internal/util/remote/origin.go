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

type Origin interface {
	fmt.Stringer
	LockedString() string

	Validate() error

	Build(digest string) *kptfilev1.Origin

	Fetch(ctx context.Context, dest string) (absPath string, digest string, err error)
	Push(ctx context.Context, dest string, kptfile *kptfilev1.KptFile) (digest string, err error)

	Ref() (string, error)
	SetRef(ref string) error
}

func NewOrigin(kf *kptfilev1.KptFile) (Origin, error) {
	const op errors.Op = "remote.NewOrigin"
	if kf != nil && kf.Origin != nil {
		switch kf.Origin.Type {
		case kptfilev1.GitOrigin:
			if kf.Upstream.Git == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin must have git information"))
			}
			u := &gitOrigin{
				git: kf.Origin.Git,
			}
			return u, nil
		case kptfilev1.OciOrigin:
			if kf.Origin.Oci == nil {
				return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin must have oci information"))
			}
			u := &ociOrigin{
				oci: kf.Origin.Oci,
			}
			return u, nil
		}
	}
	return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile origin type must be one of: %s,%s", kptfilev1.GitOrigin, kptfilev1.OciOrigin))
}
