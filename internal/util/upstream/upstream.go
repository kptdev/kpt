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
		switch kf.Upstream.Type{
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
