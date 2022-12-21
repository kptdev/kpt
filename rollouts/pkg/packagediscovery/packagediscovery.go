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

package packagediscovery

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	coreapi "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PackageDiscovery struct {
	config    gitopsv1alpha1.PackagesConfig
	client    client.Client
	namespace string
}

type DiscoveredPackage struct {
	Org       string
	Repo      string
	Directory string
	Revision  string
}

func NewPackageDiscovery(config gitopsv1alpha1.PackagesConfig, client client.Client, namespace string) *PackageDiscovery {
	return &PackageDiscovery{
		config:    config,
		client:    client,
		namespace: namespace,
	}
}

func (d *PackageDiscovery) GetPackages(ctx context.Context) ([]DiscoveredPackage, error) {
	gitRepoSelector := d.config.Git.GitRepoSelector
	gitClient, err := d.getGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create git client: %w", err)
	}

	tree, _, err := gitClient.Git.GetTree(ctx, gitRepoSelector.Org, gitRepoSelector.Repo, gitRepoSelector.Revision, true)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch tree from git: %w", err)
	}

	allPaths := []string{}
	for _, entry := range tree.Entries {
		if *entry.Type == "tree" {
			allPaths = append(allPaths, *entry.Path)
		}
	}

	packagesPaths := filterDirectories(gitRepoSelector.Directory, allPaths)

	discoveredPackages := []DiscoveredPackage{}

	for _, path := range packagesPaths {
		thisDiscoveredPackage := DiscoveredPackage{Org: gitRepoSelector.Org, Repo: gitRepoSelector.Repo, Revision: gitRepoSelector.Revision, Directory: path}
		discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
	}

	return discoveredPackages, nil
}

func (d *PackageDiscovery) getGitHubClient(ctx context.Context) (*github.Client, error) {
	gitRepoSelector := d.config.Git.GitRepoSelector

	httpClient := &http.Client{}

	if secretName := gitRepoSelector.SecretRef.Name; secretName != "" {
		var repositorySecret coreapi.Secret
		key := client.ObjectKey{Namespace: d.namespace, Name: secretName}
		if err := d.client.Get(ctx, key, &repositorySecret); err != nil {
			return nil, fmt.Errorf("cannot retrieve git credentials %s: %v", key, err)
		}

		accessToken := string(repositorySecret.Data["password"])

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)

		httpClient = oauth2.NewClient(ctx, ts)
	}

	gitClient := github.NewClient(httpClient)

	return gitClient, nil
}

func filterDirectories(pattern string, directories []string) []string {
	filteredDirectories := []string{}

	for _, directory := range directories {
		if isMatch, _ := filepath.Match(pattern, directory); isMatch {
			filteredDirectories = append(filteredDirectories, directory)
		}
	}

	return filteredDirectories
}
