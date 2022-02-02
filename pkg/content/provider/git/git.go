// Copyright 2021 Google LLC
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

package git

import (
	"context"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/git"
	"github.com/GoogleContainerTools/kpt/internal/util/remote"
	"github.com/GoogleContainerTools/kpt/pkg/content"
	"github.com/GoogleContainerTools/kpt/pkg/content/extensions"
	"github.com/GoogleContainerTools/kpt/pkg/location"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type gitProvider struct {
	repoSpec *git.RepoSpec
}

var _ content.Content = &gitProvider{}
var _ extensions.FileSystemProvider = &gitProvider{}

func Open(ctx context.Context, ref location.Git) (_ content.Content, _ location.ReferenceLock, _err error) {

	repoSpec := &git.RepoSpec{
		OrgRepo: ref.Repo,
		Path:    ref.Directory,
		Ref:     ref.Ref,
	}

	if err := remote.ClonerUsingGitExec(ctx, repoSpec); err != nil {
		return nil, nil, err
	}
	defer func() {
		if _err != nil {
			os.RemoveAll(repoSpec.Dir)
		}
	}()

	// Make lock reference for commit
	lock, err := ref.SetLock(repoSpec.Commit)
	if err != nil {
		return nil, nil, err
	}

	return &gitProvider{
		repoSpec: repoSpec,
	}, lock, nil
}

func (p *gitProvider) Close() error {
	return os.RemoveAll(p.repoSpec.Dir)
}

func (p *gitProvider) ProvideFileSystem() (filesys.FileSystem, string, error) {
	return filesys.MakeFsOnDisk(), filepath.Join(p.repoSpec.Dir, p.repoSpec.Path), nil
}
