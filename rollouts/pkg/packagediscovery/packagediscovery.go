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
	"regexp"
	"strings"

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
	gitClient, err := d.getGitHubClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create git client: %w", err)
	}

	discoveredPackages := []DiscoveredPackage{}

	repositoryNames, err := d.getRepositoryNames(gitClient, ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get repositories: %w", err)
	}

	for _, repositoryName := range repositoryNames {
		repoPackages, err := d.getPackagesForRepository(gitClient, ctx, repositoryName)
		if err != nil {
			return nil, fmt.Errorf("unable to get packages: %w", err)
		}

		discoveredPackages = append(discoveredPackages, repoPackages...)
	}

	return discoveredPackages, nil
}

func (d *PackageDiscovery) getRepositoryNames(gitClient *github.Client, ctx context.Context) ([]string, error) {
	selector := d.config.Git.Selector
	repositoryNames := []string{}

	if isSelectorField(selector.Repo) {
		// TOOD: add pagination
		listOptions := github.RepositoryListOptions{}
		listOptions.PerPage = 150

		repositories, _, err := gitClient.Repositories.List(ctx, selector.Org, &listOptions)
		if err != nil {
			return nil, err
		}

		allRepositoryNames := []string{}
		for _, repository := range repositories {
			allRepositoryNames = append(allRepositoryNames, *repository.Name)
		}

		matchRepositoryNames := filterByPattern(selector.Repo, allRepositoryNames)

		repositoryNames = append(repositoryNames, matchRepositoryNames...)
	} else {
		repositoryNames = append(repositoryNames, selector.Repo)
	}

	return repositoryNames, nil
}

func (d *PackageDiscovery) getPackagesForRepository(gitClient *github.Client, ctx context.Context, repoName string) ([]DiscoveredPackage, error) {
	discoveredPackages := []DiscoveredPackage{}
	selector := d.config.Git.Selector

	if isSelectorField(selector.Directory) {
		tree, _, err := gitClient.Git.GetTree(ctx, selector.Org, repoName, selector.Revision, true)
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
			thisDiscoveredPackage := DiscoveredPackage{Org: selector.Org, Repo: repoName, Revision: selector.Revision, Directory: directory}
			discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
		}
	} else {
		thisDiscoveredPackage := DiscoveredPackage{Org: selector.Org, Repo: repoName, Revision: selector.Revision, Directory: selector.Directory}
		discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
	}

	return discoveredPackages, nil
}

func (d *PackageDiscovery) getGitHubClient(ctx context.Context) (*github.Client, error) {
	selector := d.config.Git.Selector

	httpClient := &http.Client{}

	if secretName := selector.SecretRef.Name; secretName != "" {
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

func filterByPattern(pattern string, list []string) []string {
	matches := []string{}

	regexPattern := getRegexPattern(pattern)

	for _, value := range list {
		if isMatch := match(regexPattern, value); isMatch {
			matches = append(matches, value)
		}
	}

	return matches
}

func getRegexPattern(pattern string) string {
	var result strings.Builder

	result.WriteString("^")
	for i, literal := range strings.Split(pattern, "*") {
		if i > 0 {
			result.WriteString("[^/]+")
		}
		result.WriteString(regexp.QuoteMeta(literal))
	}
	result.WriteString("$")

	return result.String()
}

func match(pattern string, value string) bool {
	result, _ := regexp.MatchString(pattern, value)
	return result
}

func isSelectorField(value string) bool {
	return strings.Contains(value, "*")
}
