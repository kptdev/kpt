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

package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
)

type clonePackageMutation struct {
	task               *api.Task
	namespace          string
	name               string // package target name
	credentialResolver repository.CredentialResolver
}

func (m *clonePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	var cloned repository.PackageResources
	var err error

	if m.task.Clone.Upstream.UpstreamRef.Name != "" {
		cloned, err = m.cloneFromRegisteredRepository(ctx)
	} else if git := m.task.Clone.Upstream.Git; git != nil {
		cloned, err = m.cloneFromGit(ctx, git)
	} else if oci := m.task.Clone.Upstream.Oci; oci != nil {
		cloned, err = m.cloneFromOci(ctx, oci)
	}

	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	// Add any pre-existing parts of the config that have not been overwritten by the clone operation.
	for k, v := range resources.Contents {
		if _, exists := cloned.Contents[k]; !exists {
			cloned.Contents[k] = v
		}
	}

	return cloned, m.task, nil
}

func (m *clonePackageMutation) cloneFromRegisteredRepository(ctx context.Context) (repository.PackageResources, error) {
	return repository.PackageResources{}, errors.New("clone from Registered Repository is not implemented")
}

func (m *clonePackageMutation) cloneFromGit(ctx context.Context, gitPackage *api.GitPackage) (repository.PackageResources, error) {
	// TODO: Cache unregistered repositories with appropriate cache eviction policy.
	// TODO: Separate low-level repository access from Repository abstraction?

	spec := configapi.GitRepository{
		Repo:      gitPackage.Repo,
		Branch:    gitPackage.Ref,
		Directory: gitPackage.Directory,
		SecretRef: configapi.SecretRef{
			Name: gitPackage.SecretRef.Name,
		},
	}

	dir, err := ioutil.TempDir("", "clone-git-package-*")
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot create temporary directory to clone Git repository: %w", err)
	}
	defer os.RemoveAll(dir)

	credentialResolver := m.credentialResolver
	r, err := git.OpenRepository(ctx, "", "", &spec, credentialResolver, dir)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot clone Git repository: %w", err)
	}

	revision, lock, err := r.GetPackage(gitPackage.Ref, gitPackage.Directory)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot find package %s@%s: %w", gitPackage.Directory, gitPackage.Ref, err)
	}

	resources, err := revision.GetResources(ctx)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot read package resources: %w", err)
	}

	contents := resources.Spec.Resources

	// Update Kptfile
	kptfile, found := contents[v1.KptFileName]
	if !found {
		return repository.PackageResources{}, fmt.Errorf("package %s@%s is not valid; missing Kptfile", gitPackage.Directory, gitPackage.Ref)
	}

	kptfile, err = kpt.UpdateUpstreamFromGit(kptfile, m.name, lock)
	if err != nil {
		return repository.PackageResources{}, err
	}

	contents[v1.KptFileName] = kptfile

	return repository.PackageResources{
		Contents: contents,
	}, nil
}

func (m *clonePackageMutation) cloneFromOci(ctx context.Context, ociPackage *api.OciPackage) (repository.PackageResources, error) {
	return repository.PackageResources{}, errors.New("clone from OCI is not implemented")
}
