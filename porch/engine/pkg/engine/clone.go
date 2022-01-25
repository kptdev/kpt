package engine

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	tempkpt "github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
)

type clonePackageMutation struct {
	task *api.Task
	name string // package target name
}

func (m *clonePackageMutation) Apply(ctx context.Context, resources repository.PackageResources) (repository.PackageResources, *api.Task, error) {
	packageURI := m.task.Clone.Upstream.Git.Repo
	if m.task.Clone.Upstream.Git.Directory != "" {
		if !strings.HasSuffix(packageURI, ".git") {
			packageURI += ".git"
		}
		packageURI += "/" + m.task.Clone.Upstream.Git.Directory
	}
	packageVersion := m.task.Clone.Upstream.Git.Ref

	// TODO: load directly from source repository
	dir, err := ioutil.TempDir("", "kpt-pkg-get-*")
	if err != nil {
		return repository.PackageResources{}, nil, err
	}
	defer os.RemoveAll(dir)

	if err := tempkpt.PkgGet(ctx, packageURI, packageVersion, filepath.Join(dir, m.name), tempkpt.PkgGetOpts{}); err != nil {
		return repository.PackageResources{}, nil, err
	}

	loaded, err := loadResourcesFromDirectory(dir)
	if err != nil {
		return repository.PackageResources{}, nil, err
	}

	return loaded, m.task, nil
}
