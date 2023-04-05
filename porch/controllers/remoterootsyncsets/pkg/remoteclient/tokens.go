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

package remoteclient

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/gcptokensource"
	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/ksaimpersonationtokensource"
	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/ksatokensource"
	"github.com/GoogleContainerTools/kpt/porch/pkg/tokenexchange/membership"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sts/v1"
	stsv1 "google.golang.org/api/sts/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// WorkloadIdentityHelper is a helper class that does the exchanges needed for workload identity.
type WorkloadIdentityHelper struct {
	// stsClient holds a client for querying STS
	stsClient *stsv1.Service

	// corev1Client is used for kubernetes impersonation
	corev1Client corev1client.CoreV1Interface

	// dynamicClient is used to check for the membership resource
	dynamicClient dynamic.Interface

	restConfig *rest.Config

	mutex      sync.Mutex
	tokenCache map[tokenCacheKey]oauth2.TokenSource
}

type tokenCacheKey struct {
	kubeServiceAccount types.NamespacedName
	gcpServiceAccount  string
}

// Init should be called before using a WorkloadIdentityHelper
func (r *WorkloadIdentityHelper) Init(restConfig *rest.Config) error {
	r.restConfig = restConfig

	// If we want to debug RBAC/Token Exchange locally...
	// restConfigImpersonate := *restConfig
	// restConfigImpersonate.Impersonate.UserName = "system:serviceaccount:configcontroller-system:rootsyncset-impersonate"
	// restConfig = &restConfigImpersonate

	corev1Client, err := corev1client.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("error building corev1 client: %w", err)
	}
	r.corev1Client = corev1Client

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("error building dynamic client: %w", err)
	}
	r.dynamicClient = dynamicClient

	// option.WithoutAuthentication because we don't want to use our credentials for the exchange
	// STS actually gives an error: googleapi: "Error 400: Request contains an invalid argument., badRequest"
	stsClient, err := sts.NewService(context.Background(), option.WithoutAuthentication())
	if err != nil {
		return fmt.Errorf("error building sts client: %w", err)
	}
	r.stsClient = stsClient

	r.tokenCache = make(map[tokenCacheKey]oauth2.TokenSource)

	return nil
}

// GetGcloudAccessTokenSource does the exchange to get a token for the specified GCP ServiceAccount.
func (r *WorkloadIdentityHelper) GetGcloudAccessTokenSource(ctx context.Context, kubeServiceAccount types.NamespacedName, gcpServiceAccount string) (oauth2.TokenSource, error) {
	key := tokenCacheKey{
		kubeServiceAccount: kubeServiceAccount,
		gcpServiceAccount:  gcpServiceAccount,
	}

	r.mutex.Lock()
	cachedTokenSource := r.tokenCache[key]
	r.mutex.Unlock()

	if cachedTokenSource != nil {
		return cachedTokenSource, nil
	}

	var workloadIdentityPool string
	var identityProvider string

	membershipConfig, err := membership.Get(ctx, r.dynamicClient)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// TODO: Cache this?
			workloadIdentityPool, identityProvider, err = r.findWorkloadIdentityPool(ctx, kubeServiceAccount)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("error fetching membership: %w", err)
		}
	} else {
		workloadIdentityPool = membershipConfig.Spec.WorkloadIdentityPool
		identityProvider = membershipConfig.Spec.IdentityProvider
	}

	impersonated := ksaimpersonationtokensource.New(r.corev1Client, kubeServiceAccount, []string{workloadIdentityPool})

	ksaToken := ksatokensource.New(r.stsClient, impersonated, workloadIdentityPool, identityProvider)

	var scopes []string
	gcpTokenSource := gcptokensource.New(gcpServiceAccount, scopes, ksaToken)

	tokenSource := oauth2.ReuseTokenSource(nil, gcpTokenSource)

	r.mutex.Lock()
	r.tokenCache[key] = tokenSource
	r.mutex.Unlock()

	return tokenSource, nil
}

func (r *WorkloadIdentityHelper) findWorkloadIdentityPool(ctx context.Context, kubeServiceAccount types.NamespacedName) (string, string, error) {
	accessToken := ""

	// First, see if we have a valid token mounted locally in our pod
	{
		const tokenFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

		tokenBytes, err := ioutil.ReadFile(tokenFilePath)
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
		impersonated := ksaimpersonationtokensource.New(r.corev1Client, kubeServiceAccount, nil /* unspecified/default audience */)

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
