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

package porch

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/registry/porch/wi"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
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
		return nil, fmt.Errorf("cannot resolve credentials in a secret %s/%s: %w", namespace, name, err)
	}

	for _, resolver := range r.resolverChain {
		cred, found, err := resolver.Resolve(ctx, secret)
		if err != nil {
			return nil, fmt.Errorf("error resolving credential: %w", err)
		}
		if found {
			return cred, nil
		}
	}
	return nil, &NoMatchingResolverError{
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
		return nil, false, nil
	}

	return &BasicAuthCredential{
		Username: string(secret.Data["username"]),
		Password: string(secret.Data["password"]),
	}, true, nil
}

type BasicAuthCredential struct {
	Username string
	Password string
}

var _ repository.Credential = &BasicAuthCredential{}

func (b *BasicAuthCredential) Valid() bool {
	return true
}

func (b *BasicAuthCredential) ToAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: string(b.Username),
		Password: string(b.Password),
	}
}

func NewGcloudWIResolver(corev1Client *corev1client.CoreV1Client, stsClient *stsv1.Service) Resolver {
	return &GcloudWIResolver{
		coreV1Client: corev1Client,
		exchanger:    wi.NewWITokenExchanger(corev1Client, stsClient),
		circuitBreaker: &circuitBreaker{
			duration:    5 * time.Second, // We always wait at least 5 seconds before trying again.
			factor:      2,               // We double the wait time for every consecutive failure.
			maxDuration: 5 * time.Minute, // Max wait time is 5 minutes.
		},
	}
}

var _ Resolver = &GcloudWIResolver{}

type GcloudWIResolver struct {
	coreV1Client   *corev1client.CoreV1Client
	exchanger      *wi.WITokenExchanger
	circuitBreaker *circuitBreaker
}

var porchKSA = types.NamespacedName{
	Name:      "porch-server",
	Namespace: "porch-system",
}

func (g *GcloudWIResolver) Resolve(ctx context.Context, secret core.Secret) (repository.Credential, bool, error) {
	if secret.Type != WorkloadIdentityAuthType {
		return nil, false, nil
	}

	var token *oauth2.Token
	err := g.circuitBreaker.do(func() error {
		var tokenErr error
		token, tokenErr = g.getToken(ctx)
		return tokenErr
	})
	if err != nil {
		return nil, true, err
	}
	return &GcloudWICredential{
		token: token,
	}, true, nil
}

func (g *GcloudWIResolver) getToken(ctx context.Context) (*oauth2.Token, error) {
	gsa, err := g.lookupGSAEmail(ctx, porchKSA.Name, porchKSA.Namespace)
	if err != nil {
		return nil, err
	}

	token, err := g.exchanger.Exchange(ctx, porchKSA, gsa)
	if err != nil {
		return nil, err
	}

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

type GcloudWICredential struct {
	token *oauth2.Token
}

var _ repository.Credential = &GcloudWICredential{}

func (b *GcloudWICredential) Valid() bool {
	timeLeft := time.Until(b.token.Expiry)
	return timeLeft > 5*time.Minute
}

func (b *GcloudWICredential) ToAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: "token", // username doesn't matter here.
		Password: string(b.token.AccessToken),
	}
}

// circuitBreaker makes sure that failing operations are retried
// using exponential backoff.
type circuitBreaker struct {
	// duration defines how long to wait after the first failure.
	duration time.Duration
	// factor is multiplied with the previous wait time after a failure.
	factor float64
	// maxDuration is the maximum wait time we require.
	maxDuration time.Duration

	open       bool
	delay      time.Duration
	expiration time.Time
	lastErr    error
}

func (cb *circuitBreaker) do(action func() error) error {
	if cb.open {
		// If the circuit breaker is open and the required amount of
		// time hasn't passed yet, we don't retry but instead return
		// and error wrapping the error from the last attempt.
		if time.Now().Before(cb.expiration) {
			return &CircuitBreakerError{
				Err: cb.lastErr,
			}
		}
	}

	err := action()
	if err != nil {
		// After an error, we determine delay before we allow
		// another attempt.
		if !cb.open {
			cb.delay = cb.duration
		} else {
			cb.delay = cb.delay * time.Duration(cb.factor)
		}
		if cb.delay > cb.maxDuration {
			cb.delay = cb.maxDuration
		}
		// Set the new expiration in the future.
		cb.expiration = time.Now().Add(cb.delay)
		cb.open = true
		cb.lastErr = err
		return err
	}
	// If the opration succeeded, reset the circuit breaker.
	cb.open = false
	cb.delay = 0
	cb.lastErr = nil
	return nil
}

type CircuitBreakerError struct {
	Err error
}

func (cbe *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker is open. Last error: %s", cbe.Err.Error())
}

func (cbe *CircuitBreakerError) Unwrap() error {
	return cbe.Err
}
