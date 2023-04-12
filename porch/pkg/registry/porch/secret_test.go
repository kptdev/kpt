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
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretName      = "secret"
	secretNamespace = "porch-system"
)

func TestCredentialResolver(t *testing.T) {
	secretNotFoundError := apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "secret")

	testCases := map[string]struct {
		readerSecret *core.Secret
		readerErr    error

		resolverCredential repository.Credential
		resolverResolved   bool
		resolverErr        error

		expectedCredential repository.Credential
		expectedErr        error
	}{
		"no secret found": {
			readerErr:   secretNotFoundError,
			expectedErr: secretNotFoundError,
		},
		"secret is of type kubernetes.io/basic-auth": {
			readerSecret: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Type: core.SecretTypeBasicAuth,
				Data: map[string][]byte{
					"username": []byte("username"),
					"password": []byte("password"),
				},
			},
			expectedCredential: &BasicAuthCredential{
				Username: "username",
				Password: "password",
			},
		},
		"no resolver for secret type": {
			readerSecret: &core.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: secretNamespace,
				},
				Type: "notSupported",
				Data: map[string][]byte{},
			},
			expectedErr: &NoMatchingResolverError{
				Type: "notSupported",
			},
		},
	}

	for tn := range testCases {
		tc := testCases[tn]
		t.Run(tn, func(t *testing.T) {
			reader := &fakeReader{
				expectedSecret: tc.readerSecret,
				expectedErr:    tc.readerErr,
			}
			credResolver := NewCredentialResolver(reader, []Resolver{
				NewBasicAuthResolver(),
				&fakeResolver{
					credential: tc.resolverCredential,
					resolved:   tc.resolverResolved,
					err:        tc.resolverErr,
				},
			})

			cred, err := credResolver.ResolveCredential(context.Background(), secretNamespace, secretName)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expectedCredential, cred)

		})
	}
}

type fakeReader struct {
	expectedSecret *core.Secret
	expectedErr    error
}

func (f *fakeReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if f.expectedErr != nil {
		return f.expectedErr
	}
	in, ok := obj.(*core.Secret)
	if !ok {
		return fmt.Errorf("object is not of type *core.Secret")
	}
	f.expectedSecret.DeepCopyInto(in)
	return nil
}

func (f *fakeReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

type fakeResolver struct {
	credential repository.Credential
	resolved   bool
	err        error
}

func (fr *fakeResolver) Resolve(ctx context.Context, secret core.Secret) (repository.Credential, bool, error) {
	return fr.credential, fr.resolved, fr.err
}

func TestCircuitBreaker(t *testing.T) {
	cb := &circuitBreaker{
		duration:    2 * time.Second,
		factor:      4,
		maxDuration: 10 * time.Second,
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	timer := time.NewTimer(5 * time.Second)

	actionCounter := 0
loop:
	for {
		select {
		case <-ticker.C:
			_ = cb.do(func() error {
				actionCounter++
				return fmt.Errorf("error")
			})
		case <-timer.C:
			ticker.Stop()
			timer.Stop()
			break loop
		}
	}
	if got, want := actionCounter, 2; got != want {
		t.Errorf("expected function to be called %d times, but got %d", want, got)
	}
}
