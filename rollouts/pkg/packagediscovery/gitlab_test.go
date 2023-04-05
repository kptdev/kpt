// Copyright 2023 The kpt Authors
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

//go:build gitlab

package packagediscovery

import (
	"context"
	"fmt"
	"os"
	"testing"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
)

type testGitLabClientMaker struct{}

func (*testGitLabClientMaker) NewGitLabClient(ctx context.Context, config gitopsv1alpha1.GitLabSource) (*gitlab.Client, error) {
	// initialize a gitlab client
	baseURL := "https://gitlab.com/"
	token, found := os.LookupEnv("GITLAB_TOKEN")
	if !found {
		return nil, fmt.Errorf("GITLAB_TOKEN environment variable must be defined")
	}
	glc, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}
	return glc, nil
}

func TestGitLabGetPackages_SingleProject(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitLab,
		GitLab: gitopsv1alpha1.GitLabSource{
			Selector: gitopsv1alpha1.GitLabSelector{
				ProjectID: "19466078",
				Directory: "b",
			},
		},
	}

	pd := &PackageDiscovery{
		gitLabClientMaker: &testGitLabClientMaker{},
	}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	if assert.NotEmpty(t, packages) {
		pkg := packages[0]
		assert.Equal(t, pkg.Directory, "b")
		assert.Equal(t, pkg.String(), "echo-deployments")
		t.Logf("package URLs: HTTP:%s SSH:%s", pkg.HTTPURL(), pkg.SSHURL())
	}
}

func TestGitLabGetPackages_MultipleDirectory(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitLab,
		GitLab: gitopsv1alpha1.GitLabSource{
			Selector: gitopsv1alpha1.GitLabSelector{
				ProjectID: "19466078",
				Directory: "*",
			},
		},
	}

	pd := &PackageDiscovery{
		gitLabClientMaker: &testGitLabClientMaker{},
	}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	got := []string{}
	wants := []string{"a", "b", "c", "namespaces"}
	for _, pkg := range packages {
		got = append(got, pkg.Directory)
	}
	if assert.NotEmpty(t, packages) {
		assert.ElementsMatch(t, got, wants)
	}
}

func TestGitLabGetPackages_MultipleProjects(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitLab,
		GitLab: gitopsv1alpha1.GitLabSource{
			Selector: gitopsv1alpha1.GitLabSelector{
				ProjectID: "*",
				Directory: "b",
			},
		},
	}

	pd := &PackageDiscovery{
		gitLabClientMaker: &testGitLabClientMaker{},
	}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	got := []string{}
	for _, pkg := range packages {
		got = append(got, pkg.String())
	}
	if assert.NotEmpty(t, packages) {
		wants := []string{"echo-deployments"}
		assert.ElementsMatch(t, got, wants)
	}
}
