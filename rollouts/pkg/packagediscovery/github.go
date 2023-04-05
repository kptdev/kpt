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

package packagediscovery

import (
	"context"
	"fmt"
	"net/http"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	coreapi "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

func (d *PackageDiscovery) NewGitHubClient(ctx context.Context, ghConfig gitopsv1alpha1.GitHubSelector) (*github.Client, error) {
	httpClient := &http.Client{}

	if secretName := ghConfig.SecretRef.Name; secretName != "" {
		var repositorySecret coreapi.Secret
		key := client.ObjectKey{Namespace: d.namespace, Name: secretName}
		if err := d.client.Get(ctx, key, &repositorySecret); err != nil {
			return nil, fmt.Errorf("cannot retrieve github credentials %s: %v", key, err)
		}

		accessToken := string(repositorySecret.Data["password"])
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)

		httpClient = oauth2.NewClient(ctx, ts)
	}

	gitHubClient := github.NewClient(httpClient)

	return gitHubClient, nil
}

func (d *PackageDiscovery) getGitHubPackages(ctx context.Context, config gitopsv1alpha1.PackagesConfig) ([]DiscoveredPackage, error) {
	discoveredPackages := []DiscoveredPackage{}
	var ghc *github.Client
	var err error
	if config.SourceType != gitopsv1alpha1.GitHub {
		return nil, fmt.Errorf("%v source type not supported yet", config.SourceType)
	}

	gitHubSelector := config.GitHub.Selector

	if d.gitHubClientMaker != nil {
		ghc, err = d.gitHubClientMaker.NewGitHubClient(ctx, gitHubSelector)
	} else {
		ghc, err = d.NewGitHubClient(ctx, gitHubSelector)
	}
	if err != nil {
		return nil, err
	}

	repos, err := d.getRepos(ghc, gitHubSelector, ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get repositories: %w", err)
	}

	for _, repo := range repos {
		repoPackages, err := d.getPackagesForRepo(ghc, ctx, gitHubSelector, repo)
		if err != nil {
			return nil, fmt.Errorf("unable to get packages: %w", err)
		}
		discoveredPackages = append(discoveredPackages, repoPackages...)
	}
	return discoveredPackages, nil
}

func (d *PackageDiscovery) getRepos(gitHubClient *github.Client, selector gitopsv1alpha1.GitHubSelector, ctx context.Context) ([]*github.Repository, error) {
	var matchingRepos []*github.Repository

	if isSelectorField(selector.Repo) {
		// TOOD: add pagination
		listOptions := github.RepositoryListOptions{}
		listOptions.PerPage = 150

		repos, _, err := gitHubClient.Repositories.List(ctx, selector.Org, &listOptions)
		if err != nil {
			return nil, err
		}

		allRepoNames := []string{}
		for _, repo := range repos {
			allRepoNames = append(allRepoNames, *repo.Name)
		}

		matchingRepoNames := filterByPattern(selector.Repo, allRepoNames)
		matchingRepoNameSet := sets.String{}
		matchingRepoNameSet.Insert(matchingRepoNames...)

		for _, repo := range repos {
			if matchingRepoNameSet.Has(*repo.Name) {
				matchingRepos = append(matchingRepos, repo)
			}
		}
	} else {
		repo, _, err := gitHubClient.Repositories.Get(ctx, selector.Org, selector.Repo)
		if err != nil {
			return nil, err
		}
		matchingRepos = append(matchingRepos, repo)
	}

	return matchingRepos, nil
}

func (d *PackageDiscovery) getPackagesForRepo(gitHubClient *github.Client, ctx context.Context, selector gitopsv1alpha1.GitHubSelector, repo *github.Repository) ([]DiscoveredPackage, error) {
	discoveredPackages := []DiscoveredPackage{}
	branch := selector.Branch
	if branch == "" {
		branch = repo.GetDefaultBranch()
	}
	if isSelectorField(selector.Directory) {
		tree, _, err := gitHubClient.Git.GetTree(ctx, selector.Org, *repo.Name, branch, true)
		if err != nil {
			return nil, err
		}

		allDirectories := []string{}
		for _, entry := range tree.Entries {
			if *entry.Type == "tree" {
				allDirectories = append(allDirectories, *entry.Path)
			}
		}

		directories := filterByPattern(selector.Directory, allDirectories)

		for _, directory := range directories {
			thisDiscoveredPackage := DiscoveredPackage{
				Revision:   selector.Revision,
				Directory:  directory,
				GitHubRepo: repo,
				Branch:     branch,
			}
			discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
		}
	} else {
		thisDiscoveredPackage := DiscoveredPackage{
			Revision:   selector.Revision,
			Directory:  selector.Directory,
			GitHubRepo: repo,
			Branch:     branch,
		}
		discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
	}

	return discoveredPackages, nil
}
