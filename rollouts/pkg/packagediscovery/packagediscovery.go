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
	"sync"
	"time"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	coreapi "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PackageDiscovery struct {
	client    client.Client
	namespace string
	mutex     sync.Mutex
	cache     *Cache
}

type DiscoveredPackage struct {
	Org       string
	Repo      string
	Directory string
	Revision  string
}

type Cache struct {
	config     gitopsv1alpha1.PackagesConfig
	packages   []DiscoveredPackage
	expiration time.Time
}

func NewPackageDiscovery(client client.Client, namespace string) *PackageDiscovery {
	return &PackageDiscovery{
		client:    client,
		namespace: namespace,
	}
}

func (d *PackageDiscovery) GetPackages(ctx context.Context, config gitopsv1alpha1.PackagesConfig) ([]DiscoveredPackage, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.useCache(config) {
		return d.cache.packages, nil
	}

	if config.SourceType != gitopsv1alpha1.GitHub {
		return nil, fmt.Errorf("%v source type not supported yet", config.SourceType)
	}

	gitHubSelector := config.GitHub.Selector

	gitHubClient, err := d.getGitHubClient(ctx, gitHubSelector)
	if err != nil {
		return nil, fmt.Errorf("unable to create github client: %w", err)
	}

	discoveredPackages := []DiscoveredPackage{}

	repositoryNames, err := d.getRepositoryNames(gitHubClient, gitHubSelector, ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get repositories: %w", err)
	}

	for _, repositoryName := range repositoryNames {
		repoPackages, err := d.getPackagesForRepository(gitHubClient, ctx, gitHubSelector, repositoryName)
		if err != nil {
			return nil, fmt.Errorf("unable to get packages: %w", err)
		}

		discoveredPackages = append(discoveredPackages, repoPackages...)
	}

	d.cache = &Cache{
		packages:   discoveredPackages,
		config:     config,
		expiration: time.Now().Add(1 * time.Minute),
	}

	return discoveredPackages, nil
}

func (d *PackageDiscovery) getRepositoryNames(gitHubClient *github.Client, selector gitopsv1alpha1.GitHubSelector, ctx context.Context) ([]string, error) {
	repositoryNames := []string{}

	if isSelectorField(selector.Repo) {
		// TOOD: add pagination
		listOptions := github.RepositoryListOptions{}
		listOptions.PerPage = 150

		repositories, _, err := gitHubClient.Repositories.List(ctx, selector.Org, &listOptions)
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

func (d *PackageDiscovery) getPackagesForRepository(gitHubClient *github.Client, ctx context.Context, selector gitopsv1alpha1.GitHubSelector, repoName string) ([]DiscoveredPackage, error) {
	discoveredPackages := []DiscoveredPackage{}

	if isSelectorField(selector.Directory) {
		tree, _, err := gitHubClient.Git.GetTree(ctx, selector.Org, repoName, selector.Revision, true)
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

func (d *PackageDiscovery) getGitHubClient(ctx context.Context, selector gitopsv1alpha1.GitHubSelector) (*github.Client, error) {
	httpClient := &http.Client{}

	if secretName := selector.SecretRef.Name; secretName != "" {
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

func (d *PackageDiscovery) useCache(config gitopsv1alpha1.PackagesConfig) bool {
	return d.cache != nil && cmp.Equal(config, d.cache.config) && time.Now().Before(d.cache.expiration)
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
