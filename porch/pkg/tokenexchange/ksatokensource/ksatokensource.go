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

package ksatokensource

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	stsv1 "google.golang.org/api/sts/v1"
)

// New returns an oauth2.TokenSource that exchanges the KSA token from ksaToken
// for a GCP access token.
func New(stsService *stsv1.Service, ksaToken oauth2.TokenSource, workloadIdentityPool, identityProvider string) oauth2.TokenSource {
	return &ksaTokenSource{
		ksaToken:             ksaToken,
		workloadIdentityPool: workloadIdentityPool,
		identityProvider:     identityProvider,
		stsService:           stsService,
	}
}

// ksaTokenSource implements oauth2.TokenSource for exchanging KSA tokens for
// GCP tokens. It can be wrapped in a ReuseTokenSource to cache tokens until
// expiry.
type ksaTokenSource struct {
	// ksaToken is the source of the kubernetes serviceaccount token.
	ksaToken oauth2.TokenSource
	// workloadIdentityPool is the Workload Identity Pool to use when exchanging the KSA
	// token for a GCP token.
	workloadIdentityPool string
	// identityProvider is the Identity Provider to use when exchanging the KSA
	// token for a GCP token.
	identityProvider string

	stsService *stsv1.Service
}

// ksaTokenSource implements oauth2.TokenSource
var _ oauth2.TokenSource = &ksaTokenSource{}

// Token exchanges a KSA token for a GCP access token, returning the GCP token.
func (ts *ksaTokenSource) Token() (*oauth2.Token, error) {
	ksaToken, err := ts.ksaToken.Token()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exchangeTime := time.Now()
	workloadIdentityPool := ts.workloadIdentityPool
	identityProvider := ts.identityProvider

	audience := fmt.Sprintf("identitynamespace:%s:%s", workloadIdentityPool, identityProvider)

	request := &stsv1.GoogleIdentityStsV1ExchangeTokenRequest{
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
		SubjectToken:       ksaToken.AccessToken,
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		Audience:           audience,
		Scope:              "https://www.googleapis.com/auth/iam",
	}

	response, err := ts.stsService.V1.Token(request).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get federated token from STS: %w", err)
	}

	token := &oauth2.Token{
		AccessToken: response.AccessToken,
		TokenType:   response.TokenType,
		Expiry:      exchangeTime.Add(time.Duration(response.ExpiresIn) * time.Second),
	}
	return token, nil

}

// ExtractIsssuer will extract the issuer field from the provided JWT token
func ExtractIssuer(jwtToken string) (string, error) {
	return ExtractJWTString(jwtToken, "iss")
}

// ExtractJWTString extracts the named field from the provided JWT token.
func ExtractJWTString(jwtToken string, key string) (string, error) {
	tokens := strings.Split(jwtToken, ".")
	if len(tokens) != 3 {
		// Don't log the token as it may be sensitive
		return "", fmt.Errorf("error getting identity provider from JWT (unexpected number of tokens)")
	}
	b, err := base64.RawURLEncoding.DecodeString(tokens[1])
	if err != nil {
		// Don't log the token as it may be sensitive
		return "", fmt.Errorf("error getting identity provider from JWT (cannot decode base64)")
	}
	m := make(map[string]interface{})
	if err := json.Unmarshal(b, &m); err != nil {
		// Don't log the token as it may be sensitive
		return "", fmt.Errorf("error getting identity provider from JWT (cannot decode json)")
	}
	val := m[key]
	if val == nil {
		// Don't log the token as it may be sensitive
		return "", fmt.Errorf("error getting identity provider from JWT (key %q not found)", key)
	}
	s, ok := val.(string)
	if !ok {
		// Don't log the token as it may be sensitive
		return "", fmt.Errorf("error getting identity provider from JWT (key %q was not string)", key)
	}
	return s, nil
}
