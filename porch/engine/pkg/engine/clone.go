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
	"strings"

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
	cad                CaDEngine
	credentialResolver repository.CredentialResolver
	referenceResolver  ReferenceResolver
}

func (m *clonePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	var cloned repository.PackageResources
	var err error

	if ref := m.task.Clone.Upstream.UpstreamRef; ref.Name != "" {
		cloned, err = m.cloneFromRegisteredRepository(ctx, ref)
	} else if git := m.task.Clone.Upstream.Git; git != nil {
		cloned, err = m.cloneFromGit(ctx, git)
	} else if oci := m.task.Clone.Upstream.Oci; oci != nil {
		cloned, err = m.cloneFromOci(ctx, oci)
	} else {
		err = errors.New("invalid clone source (neither of git, oci, nor upstream were specified)")
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

func (m *clonePackageMutation) cloneFromRegisteredRepository(ctx context.Context, ref api.PackageRevisionRef) (repository.PackageResources, error) {
	parsed, err := parseUpstreamRef(ref.Name)
	if err != nil {
		return repository.PackageResources{}, err
	}
	var resolved configapi.Repository
	if err := m.referenceResolver.ResolveReference(ctx, m.namespace, parsed.repo, &resolved); err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot find repository %s/%s: %w", m.namespace, parsed.repo, err)
	}

	open, err := m.cad.OpenRepository(ctx, &resolved)
	if err != nil {
		return repository.PackageResources{}, err
	}

	revisions, err := open.ListPackageRevisions(ctx)
	if err != nil {
		return repository.PackageResources{}, err
	}

	var revision repository.PackageRevision
	for _, rev := range revisions {
		if rev.Name() == ref.Name {
			revision = rev
			break
		}
	}
	if revision == nil {
		return repository.PackageResources{}, fmt.Errorf("cannot find package revision %q", ref.Name)
	}

	resources, err := revision.GetResources(ctx)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot read contents of package %q: %w", ref.Name, err)
	}

	upstream, lock, err := revision.GetUpstreamLock()
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot determine upstream lock for package %q: %w", ref.Name, err)
	}

	// Update Kptfile
	if err := kpt.UpdateKptfileUpstream(m.name, resources.Spec.Resources, upstream, lock); err != nil {
		return repository.PackageResources{}, fmt.Errorf("failed to apply upstream lock to pakcage %q: %w", ref.Name, err)
	}

	return repository.PackageResources{
		Contents: resources.Spec.Resources,
	}, nil
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
	if err := kpt.UpdateKptfileUpstream(m.name, contents, v1.Upstream{
		Type: v1.GitOrigin,
		Git: &v1.Git{
			Repo:      lock.Repo,
			Directory: lock.Directory,
			Ref:       lock.Ref,
		},
	}, v1.UpstreamLock{
		Type: v1.GitOrigin,
		Git:  &lock,
	}); err != nil {
		return repository.PackageResources{}, fmt.Errorf("failed to clone package %s@%s: %w", gitPackage.Directory, gitPackage.Ref, err)
	}

	return repository.PackageResources{
		Contents: contents,
	}, nil
}

func (m *clonePackageMutation) cloneFromOci(ctx context.Context, ociPackage *api.OciPackage) (repository.PackageResources, error) {
	return repository.PackageResources{}, errors.New("clone from OCI is not implemented")
}

type parsedRef struct {
	repo, pkg, version string
}

func parseUpstreamRef(ref string) (parsedRef, error) {
	first, last := strings.Index(ref, ":"), strings.LastIndex(ref, ":")
	if first == last {
		return parsedRef{}, fmt.Errorf("invalid package name %q", ref)
	}
	return parsedRef{repo: ref[:first], pkg: ref[first+1 : last], version: ref[last+1:]}, nil
}
