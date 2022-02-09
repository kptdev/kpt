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

	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewCredentialResolver(coreClient client.Reader) repository.CredentialResolver {
	return &secretResolver{
		coreClient: coreClient,
	}
}

type secretResolver struct {
	coreClient client.Reader
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

	return repository.Credential{
		Data: secret.Data,
	}, nil
}
