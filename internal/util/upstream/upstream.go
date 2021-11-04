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
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

type Fetcher interface {
	fmt.Stringer
	Validate() error
	ApplyUpstream(kf *kptfilev1.KptFile)
	FetchUpstream(ctx context.Context, dest string) error
}

func NewUpstream(kf *kptfilev1.KptFile) (Fetcher, error) {
	const op errors.Op = "upstream.NewUpstreamFetcher"
	if kf != nil && kf.Upstream != nil {
		switch kf.Upstream.Type {
		case kptfilev1.GitOrigin:
			if kf.Upstream.Git != nil {
				return &gitUpstream{
					git: kf.Upstream.Git,
				}, nil
			}
		case kptfilev1.OciOrigin:
			if kf.Upstream.Oci != nil {
				return &ociUpstream{
					image: kf.Upstream.Oci.Image,
				}, nil
			}
		}
	}
	return nil, errors.E(op, errors.MissingParam, fmt.Errorf("kptfile upstream type must be one of: %s,%s", kptfilev1.GitOrigin, kptfilev1.OciOrigin))
}
