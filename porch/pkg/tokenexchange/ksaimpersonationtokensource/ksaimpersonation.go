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

package ksaimpersonationtokensource

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

// New returns an oauth2.TokenSource that exchanges the KSA token at ksaTokenPath
// for a GCP access token.
func New(corev1Client corev1client.CoreV1Interface, serviceAccount types.NamespacedName, audiences []string) oauth2.TokenSource {
	return &ksaImpersonationTokenSource{
		corev1Client:   corev1Client,
		serviceAccount: serviceAccount,
		audiences:      audiences,
	}
}

// ksaImpersonationTokenSource implements oauth2.TokenSource for exchanging KSA tokens for
// GCP tokens. It can be wrapped in a ReuseTokenSource to cache tokens until
// expiry.
type ksaImpersonationTokenSource struct {
	corev1Client corev1client.CoreV1Interface

	// serviceAccount is the name of the serviceAccount to impersonate
	serviceAccount types.NamespacedName

	// audiences is the set of audiences to request
	audiences []string
}

// ksaTokenSource implements oauth2.TokenSource
var _ oauth2.TokenSource = &ksaImpersonationTokenSource{}

// Token exchanges a KSA token for a GCP access token, returning the GCP token.
func (ts *ksaImpersonationTokenSource) Token() (*oauth2.Token, error) {
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: ts.audiences,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	klog.V(2).Infof("getting token for kubernetes serviceaccount %v", ts.serviceAccount)
	response, err := ts.corev1Client.ServiceAccounts(ts.serviceAccount.Namespace).CreateToken(ctx, ts.serviceAccount.Name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to CreateToken for %s: %w", ts.serviceAccount, err)
	}

	exchangeTime := time.Now()

	serviceAccountToken := &oauth2.Token{
		AccessToken: response.Status.Token,
		TokenType:   "Bearer",
	}

	if response.Spec.ExpirationSeconds != nil {
		serviceAccountToken.Expiry = exchangeTime.Add(time.Duration(*response.Spec.ExpirationSeconds) * time.Second)
	} else {
		klog.Warningf("service account token did not include expirationSeconds")
		serviceAccountToken.Expiry = exchangeTime
	}

	return serviceAccountToken, nil
}
