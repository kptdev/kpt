package engine

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/kpt/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/git"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
)

type clonePackageMutation struct {
	task *api.Task
	name string // package target name
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
	return repository.PackageResources{}, errors.New("Clone from Registered Repository is not implemented")
}

func (m *clonePackageMutation) cloneFromGit(ctx context.Context, gitPackage *api.GitPackage) (repository.PackageResources, error) {
	// TODO: Cache unregistered repositories with appropriate cache eviction policy.
	// TODO: Separate low-level repository access from Repository abstraction?

	spec := configapi.GitRepository{
		Repo:      gitPackage.Repo,
		Branch:    gitPackage.Ref,
		Directory: gitPackage.Directory,
	}

	dir, err := ioutil.TempDir("", "clone-git-package-*")
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot create temporary directory to clone Git repository: %w", err)
	}
	defer os.RemoveAll(dir)

	// TODO: Add support for authentication.
	var auth repository.AuthOptions = nil
	r, err := git.OpenRepository("", "", &spec, auth, dir)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot clone Git repository: %w", err)
	}

	revision, lock, err := r.GetPackage(gitPackage.Ref, gitPackage.Directory)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot find packge %s@%s: %w", gitPackage.Directory, gitPackage.Ref, err)
	}

	resources, err := revision.GetResources(ctx)
	if err != nil {
		return repository.PackageResources{}, fmt.Errorf("cannot read package resources: %w", err)
	}

	// Rewrite paths
	results := map[string]string{}
	prefix := gitPackage.Directory + "/"
	for k, v := range resources.Spec.Resources {
		if !strings.HasPrefix(k, prefix) {
			return repository.PackageResources{}, fmt.Errorf("invalid file path within a package: %q", k)
		}
		results[path.Join(m.name, k[len(prefix):])] = v
	}

	// Update Kptfile
	kptfilePath := path.Join(m.name, "Kptfile")
	kptfile, found := results[kptfilePath]
	if !found {
		return repository.PackageResources{}, fmt.Errorf("package %s@%s is not valid; missing Kptfile", gitPackage.Directory, gitPackage.Ref)
	}

	kptfile, err = kpt.UpdateUpstreamFromGit(kptfile, m.name, lock)
	if err != nil {
		return repository.PackageResources{}, err
	}

	results[kptfilePath] = kptfile

	return repository.PackageResources{
		Contents: results,
	}, nil
}

func (m *clonePackageMutation) cloneFromOci(ctx context.Context, ociPackage *api.OciPackage) (repository.PackageResources, error) {
	return repository.PackageResources{}, errors.New("Clone from OCI is not implemented")
}
