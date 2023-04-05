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

//go:build github

package packagediscovery

import (
	"context"
	"testing"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestGitHubGetPackages_SingleRepo(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitHub,
		GitHub: gitopsv1alpha1.GitHubSource{
			Selector: gitopsv1alpha1.GitHubSelector{
				Org:       "droot",
				Repo:      "store",
				Directory: "namespaces",
			},
		},
	}

	pd := &PackageDiscovery{}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	if assert.NotEmpty(t, packages) {
		pkg := packages[0]
		assert.Equal(t, pkg.Directory, "namespaces")
		assert.Equal(t, pkg.String(), "store")
		t.Logf("package URLs: HTTP:%s SSH:%s", pkg.HTTPURL(), pkg.SSHURL())
	}
}

func TestGitHubGetPackages_MultipleDirectory(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitHub,
		GitHub: gitopsv1alpha1.GitHubSource{
			Selector: gitopsv1alpha1.GitHubSelector{
				Org:       "droot",
				Repo:      "echo-deployments",
				Directory: "store-*",
			},
		},
	}
	pd := &PackageDiscovery{}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	want := []string{"store-1", "store-2", "store-3", "store-4", "store-5"}
	got := []string{}
	for _, pkg := range packages {
		got = append(got, pkg.Directory)
	}
	if assert.NotEmpty(t, packages) {
		assert.ElementsMatch(t, got, want)
	}
}

func TestGitHubGetPackages_MultipleRepos(t *testing.T) {
	config := gitopsv1alpha1.PackagesConfig{
		SourceType: gitopsv1alpha1.GitHub,
		GitHub: gitopsv1alpha1.GitHubSource{
			Selector: gitopsv1alpha1.GitHubSelector{
				Org:  "droot",
				Repo: "store-*",
			},
		},
	}
	pd := &PackageDiscovery{}
	packages, err := pd.GetPackages(context.Background(), config)
	assert.NoError(t, err)
	want := []string{"store-1", "store-2", "store-3", "store-4", "store-5"}
	got := []string{}
	for _, pkg := range packages {
		got = append(got, pkg.String())
	}
	if assert.NotEmpty(t, packages) {
		assert.ElementsMatch(t, got, want)
	}
}
