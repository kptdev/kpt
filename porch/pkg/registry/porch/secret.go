// Copyright 2022 Google LLC
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

package porch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/registry/porch/wi"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"golang.org/x/oauth2"
	stsv1 "google.golang.org/api/sts/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Values for scret types supported by porch.
	BasicAuthType            = core.SecretTypeBasicAuth
	WorkloadIdentityAuthType = "kpt.dev/workload-identity-auth"

	// Annotation used to specify the gsa for a ksa.
	WIGCPSAAnnotation = "iam.gke.io/gcp-service-account"
)

func NewCredentialResolver(coreClient client.Reader, resolverChain []Resolver) repository.CredentialResolver {
	return &secretResolver{
		coreClient:    coreClient,
		resolverChain: resolverChain,
	}
}

type secretResolver struct {
	resolverChain []Resolver
	coreClient    client.Reader
}

type Resolver interface {
	Resolve(ctx context.Context, secret core.Secret) (repository.Credential, bool, error)
}

var _ repository.CredentialResolver = &secretResolver{}

func (r *secretResolver) ResolveCredential(ctx context.Context, namespace, name string) (repository.Credential, error) {
	var secret core.Secret
	if err := r.coreClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &secret); err != nil {
		return repository.Credential{}, fmt.Errorf("cannot resolve credentials in a secret %s/%s: %w", namespace, name, err)
	}

	for _, resolver := range r.resolverChain {
		cred, found, err := resolver.Resolve(ctx, secret)
		if err != nil {
			return repository.Credential{}, fmt.Errorf("error resolving credential: %w", err)
		}
		if found {
			return cred, nil
		}
	}
	return repository.Credential{}, &NoMatchingResolverError{
		Type: string(secret.Type),
	}
}

type NoMatchingResolverError struct {
	Type string
}

func (e *NoMatchingResolverError) Error() string {
	return fmt.Sprintf("no resolver for secret with type %s", e.Type)
}

func (e *NoMatchingResolverError) Is(err error) bool {
	nmre, ok := err.(*NoMatchingResolverError)
	if !ok {
		return false
	}
	return nmre.Type == e.Type
}

func NewBasicAuthResolver() Resolver {
	return &BasicAuthResolver{}
}

var _ Resolver = &BasicAuthResolver{}

type BasicAuthResolver struct{}

func (b *BasicAuthResolver) Resolve(_ context.Context, secret core.Secret) (repository.Credential, bool, error) {
	if secret.Type != BasicAuthType {
		return repository.Credential{}, false, nil
	}

	return repository.Credential{
		Data: secret.Data,
	}, true, nil
}

func NewGcloudWIResolver(corev1Client *corev1client.CoreV1Client, stsClient *stsv1.Service) Resolver {
	return &GcloudWIResolver{
		coreV1Client: corev1Client,
		exchanger:    wi.NewWITokenExchanger(corev1Client, stsClient),
		tokenCache:   make(map[tokenCacheKey]*oauth2.Token),
	}
}

var _ Resolver = &GcloudWIResolver{}

type GcloudWIResolver struct {
	coreV1Client *corev1client.CoreV1Client
	exchanger    *wi.WITokenExchanger

	mutex      sync.Mutex
	tokenCache map[tokenCacheKey]*oauth2.Token
}

type tokenCacheKey struct {
	ksa types.NamespacedName
	gsa string
}

var porchKSA = types.NamespacedName{
	Name:      "porch-server",
	Namespace: "porch-system",
}

func (g *GcloudWIResolver) Resolve(ctx context.Context, secret core.Secret) (repository.Credential, bool, error) {
	if secret.Type != WorkloadIdentityAuthType {
		return repository.Credential{}, false, nil
	}

	token, err := g.getToken(ctx)
	if err != nil {
		return repository.Credential{}, true, err
	}
	return repository.Credential{
		Data: map[string][]byte{
			"username": []byte("token"), // username doesn't matter here.
			"password": []byte(token.AccessToken),
		},
	}, true, nil
}

func (g *GcloudWIResolver) getToken(ctx context.Context) (*oauth2.Token, error) {
	gsa, err := g.lookupGSAEmail(ctx, porchKSA.Name, porchKSA.Namespace)
	if err != nil {
		return nil, err
	}

	tokenKey := tokenCacheKey{
		ksa: porchKSA,
		gsa: gsa,
	}

	g.mutex.Lock()
	token, found := g.tokenCache[tokenKey]
	g.mutex.Unlock()

	if found {
		timeLeft := time.Until(token.Expiry)
		if timeLeft > 5*time.Minute {
			return token, nil
		}
	}

	token, err = g.exchanger.Exchange(ctx, porchKSA, gsa)
	if err != nil {
		return nil, err
	}

	g.mutex.Lock()
	g.tokenCache[tokenKey] = token
	g.mutex.Unlock()

	return token, nil
}

func (g *GcloudWIResolver) lookupGSAEmail(ctx context.Context, name, namespace string) (string, error) {
	sa, err := g.coreV1Client.ServiceAccounts(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", fmt.Errorf("porch service account %s/%s not found", namespace, name)
		}
		return "", fmt.Errorf("error looking up porch service account %s/%s: %w", namespace, name, err)
	}
	gsa, found := sa.Annotations[WIGCPSAAnnotation]
	if !found {
		return "", fmt.Errorf("%s annotation not found on porch sa", WIGCPSAAnnotation)
	}
	return gsa, nil
}
