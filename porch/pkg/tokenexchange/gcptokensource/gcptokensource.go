// Copyright 2022 The kpt Authors
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

package gcptokensource

import (
	"context"
	"fmt"
	"time"

	iamv1 "cloud.google.com/go/iam/credentials/apiv1"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/credentials/v1"
	"k8s.io/klog/v2"
)

// New returns an oauth2.TokenSource that exchanges tokens from ts for tokens
// that authenticate as GCP Service Accounts.
func New(gcpServiceAccount string, scopes []string, tokenSource oauth2.TokenSource) oauth2.TokenSource {
	// The cloud-platform scope is always required for the token exchange.
	scopes = append(scopes, "https://www.googleapis.com/auth/cloud-platform")
	return &gcpTokenSource{
		gcpServiceAccount: gcpServiceAccount,
		scopes:            scopes,
		tokenSource:       tokenSource,
	}
}

// gcpTokenSource produces tokens that authenticate as GCP ServiceAccounts.
type gcpTokenSource struct {
	gcpServiceAccount string
	scopes            []string
	tokenSource       oauth2.TokenSource
}

// ensure gcpTokenSource implements oauth2.TokenSource
var _ oauth2.TokenSource = &gcpTokenSource{}

// Token exchanges the input token for a GCP SA token.
func (ts *gcpTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// use the provided token source to make the request
	c, err := iamv1.NewIamCredentialsClient(ctx, option.WithTokenSource(ts.tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to construct IAM client: %w", err)
	}
	resp, err := c.GenerateAccessToken(ctx,
		&iampb.GenerateAccessTokenRequest{
			Name:  "projects/-/serviceAccounts/" + ts.gcpServiceAccount,
			Scope: ts.scopes,
		})
	if err != nil {
		return nil, fmt.Errorf("token exchange for GCP serviceaccount %q failed: %w", ts.gcpServiceAccount, err)
	}

	klog.Infof("got GCP token for %v", ts.gcpServiceAccount)

	expiry, err := ptypes.Timestamp(resp.ExpireTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expire time on returned token: %w", err)
	}
	return &oauth2.Token{
		AccessToken: resp.AccessToken,
		Expiry:      expiry,
	}, nil
}
