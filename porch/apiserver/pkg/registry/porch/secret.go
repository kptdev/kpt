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

	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func resolveRepositorySecret(ctx context.Context, coreClient client.Reader, spec *configapi.Repository) (map[string][]byte, error) {
	var secretName string

	switch spec.Spec.Type {
	case configapi.RepositoryTypeOCI:
		oci := spec.Spec.Oci
		if oci != nil {
			secretName = oci.SecretRef.Name
		}

	case configapi.RepositoryTypeGit:
		git := spec.Spec.Git
		if git != nil {
			secretName = git.SecretRef.Name
		}

	default:
		return nil, fmt.Errorf("unrecognized repository type: %q", spec.Spec.Type)
	}

	if secretName == "" {
		return nil, nil
	}

	var secret core.Secret
	if err := coreClient.Get(ctx, client.ObjectKey{
		Namespace: spec.Namespace,
		Name:      secretName,
	}, &secret); err != nil {
		return nil, err
	}

	return secret.Data, nil
}
