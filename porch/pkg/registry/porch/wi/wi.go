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

package wi

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/gcptokensource"
	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/ksaimpersonationtokensource"
	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/ksatokensource"
	"golang.org/x/oauth2"
	stsv1 "google.golang.org/api/sts/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/klog/v2"
)

func NewWITokenExchanger(corev1Client *corev1client.CoreV1Client, stsClient *stsv1.Service) *WITokenExchanger {
	return &WITokenExchanger{
		corev1Client: corev1Client,
		stsClient:    stsClient,
	}
}

type WITokenExchanger struct {
	corev1Client *corev1client.CoreV1Client
	stsClient    *stsv1.Service
}

func (w *WITokenExchanger) Exchange(ctx context.Context, ksa types.NamespacedName, gsa string) (*oauth2.Token, error) {
	workloadIdentityPool, identityProvider, err := w.findWorkloadIdentityPool(ctx, ksa)
	if err != nil {
		return nil, err
	}

	impersonated := ksaimpersonationtokensource.New(w.corev1Client, ksa, []string{workloadIdentityPool})

	ksaToken := ksatokensource.New(w.stsClient, impersonated, workloadIdentityPool, identityProvider)

	var scopes []string
	gcpToken := gcptokensource.New(gsa, scopes, ksaToken)

	token, err := gcpToken.Token()
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %w", err)
	}

	return token, nil
}

func (w *WITokenExchanger) findWorkloadIdentityPool(ctx context.Context, kubeServiceAccount types.NamespacedName) (string, string, error) {
	accessToken := ""

	// First, see if we have a valid token mounted locally in our pod
	{
		const tokenFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

		tokenBytes, err := os.ReadFile(tokenFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				klog.V(2).Infof("token file not found at %q", tokenFilePath)
			} else {
				klog.Warningf("error reading token file from %q: %v", tokenFilePath, err)
			}
		} else {
			klog.Infof("found token at %q", tokenFilePath)
			accessToken = string(tokenBytes)
		}
	}

	// We could also query the kube apiserver at /.well-known/openid-configuration
	// kubectl get --raw /.well-known/openid-configuration
	// {"issuer":"https://container.googleapis.com/v1/projects/example-project-id/locations/us-central1/clusters/krmapihost-control","jwks_uri":"https://172.16.0.130:443/openid/v1/jwks","response_types_supported":["id_token"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}

	if accessToken == "" {
		// We get a token for our own service account, so we can extract the issuer
		klog.Infof("token not found at well-known path, requesting token from apiserver")
		impersonated := ksaimpersonationtokensource.New(w.corev1Client, kubeServiceAccount, nil /* unspecified/default audience */)

		token, err := impersonated.Token()
		if err != nil {
			return "", "", fmt.Errorf("failed to get kube token for %s: %w", kubeServiceAccount, err)
		} else {
			accessToken = token.AccessToken
		}
	}

	issuer, err := ksatokensource.ExtractIssuer(accessToken)
	if err != nil {
		return "", "", err
	}

	if strings.HasPrefix(issuer, "https://container.googleapis.com/") {
		path := strings.TrimPrefix(issuer, "https://container.googleapis.com/")
		tokens := strings.Split(path, "/")
		for i := 0; i+1 < len(tokens); i++ {
			if tokens[i] == "projects" {
				workloadIdentityPool := tokens[i+1] + ".svc.id.goog"
				klog.Infof("inferred workloadIdentityPool as %q", workloadIdentityPool)
				return workloadIdentityPool, issuer, nil
			}
		}
		return "", "", fmt.Errorf("could not extract project from issue %q", issuer)
	} else {
		return "", "", fmt.Errorf("unknown issuer %q", issuer)
	}
}
