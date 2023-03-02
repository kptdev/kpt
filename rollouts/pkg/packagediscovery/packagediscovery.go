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
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
	coreapi "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

type PackageDiscovery struct {
	client            client.Client
	namespace         string
	mutex             sync.Mutex
	cache             *Cache
	gitLabClientMaker GitLabClientMaker
	gitHubClientMaker GitHubClientMaker
}

type GitLabClientMaker interface {
	NewGitLabClient(ctx context.Context, config gitopsv1alpha1.GitLabSource) (*gitlab.Client, error)
}

type GitHubClientMaker interface {
	NewGitHubClient(ctx context.Context, config gitopsv1alpha1.GitHubSelector) (*github.Client, error)
}

type DiscoveredPackage struct {
	Org       string
	Repo      string
	Directory string
	Revision  string
	// GitLabProject contains the package info retrieved from GitLab
	GitLabProject *gitlab.Project
	// GithubRepo contains the package info retrieved from GitHub
	GitHubRepo *github.Repository
}

func (dp *DiscoveredPackage) HTTPURL() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.HTTPURLToRepo
	case dp.GitHubRepo != nil:
		return dp.GitHubRepo.GetCloneURL()
	}
	return ""
}

func (dp *DiscoveredPackage) SSHURL() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.SSHURLToRepo
	case dp.GitHubRepo != nil:
		return dp.GitHubRepo.GetSSHURL()
	}
	return ""
}

func (dp *DiscoveredPackage) String() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.String()
	case dp.GitHubRepo != nil:
		return dp.GitHubRepo.String()
	}
	return ""
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

func (d *PackageDiscovery) GetPackages(ctx context.Context, config gitopsv1alpha1.PackagesConfig) (discoveredPackages []DiscoveredPackage, err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.useCache(config) {
		return d.cache.packages, nil
	}

	switch config.SourceType {
	case gitopsv1alpha1.GitHub:
		discoveredPackages, err = d.getGitHubPackages(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch github packages: %w", err)
		}
	case gitopsv1alpha1.GitLab:
		discoveredPackages, err = d.getGitLabPackages(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch gitlab packages: %w", err)
		}
	default:
		return nil, fmt.Errorf("%v source type not supported yet", config.SourceType)
	}

	d.cache = &Cache{
		packages:   discoveredPackages,
		config:     config,
		expiration: time.Now().Add(1 * time.Minute),
	}

	return discoveredPackages, nil
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

	if isSelectorField(selector.Directory) {
		tree, _, err := gitHubClient.Git.GetTree(ctx, selector.Org, *repo.Name, repo.GetDefaultBranch(), true)
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
				Org:        selector.Org,
				Repo:       *repo.Name,
				Revision:   selector.Revision,
				Directory:  directory,
				GitHubRepo: repo,
			}
			discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
		}
	} else {
		thisDiscoveredPackage := DiscoveredPackage{
			Org:        selector.Org,
			Repo:       *repo.Name,
			Revision:   selector.Revision,
			Directory:  selector.Directory,
			GitHubRepo: repo,
		}
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

func (d *PackageDiscovery) NewGitLabClient(ctx context.Context, gitlabSource gitopsv1alpha1.GitLabSource) (*gitlab.Client, error) {

	// initialize a gitlab client
	secretName := gitlabSource.SecretRef.Name
	if secretName == "" {
		return nil, fmt.Errorf("GitLab secret reference is missing from the config")
	}
	var repositorySecret coreapi.Secret
	key := client.ObjectKey{
		Namespace: d.namespace,
		Name:      secretName,
	}
	if err := d.client.Get(ctx, key, &repositorySecret); err != nil {
		return nil, fmt.Errorf("cannot retrieve gitlab credentials %s: %v", key, err)
	}

	accessToken := string(repositorySecret.Data["token"])
	// TODO(droot): BaseURL should also be configurable through the API
	baseURL := "https://gitlab.com/"

	glc, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}
	return glc, nil
}

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

// getGitLabPackages looks up GitLab for packages specified by a given package selector.
func (d *PackageDiscovery) getGitLabPackages(ctx context.Context, config gitopsv1alpha1.PackagesConfig) ([]DiscoveredPackage, error) {
	var discoveredPackages []DiscoveredPackage
	var glc *gitlab.Client
	var err error

	if d.gitLabClientMaker != nil {
		glc, err = d.gitLabClientMaker.NewGitLabClient(ctx, config.GitLab)
	} else {
		glc, err = d.NewGitLabClient(ctx, config.GitLab)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}
	gitlabSelector := config.GitLab.Selector

	projects, err := d.getGitLabProjects(ctx, glc, gitlabSelector)
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		repoPackages, err := d.getGitLabPackagesForProject(ctx, glc, project, gitlabSelector)
		if err != nil {
			return nil, err
		}
		discoveredPackages = append(discoveredPackages, repoPackages...)
	}
	return discoveredPackages, nil
}

// getGitLabPackages looks up GitLab for packages specified by a given package selector.
func (d *PackageDiscovery) getGitLabProjects(ctx context.Context, glc *gitlab.Client, selector gitopsv1alpha1.GitLabSelector) ([]*gitlab.Project, error) {
	var matchingProjects []*gitlab.Project

	if isSelectorField(selector.ProjectID) {
		membershipAccess := true
		options := &gitlab.ListProjectsOptions{
			// TODO: support pagination
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 150,
			},
			Membership: gitlab.Bool(membershipAccess),
		}
		projects, _, err := glc.Projects.ListProjects(options)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch gitlab projects: %w", err)
		}
		allRepoNames := []string{}
		for _, project := range projects {
			allRepoNames = append(allRepoNames, project.Name)
		}

		matchingRepoNames := filterByPattern(selector.ProjectID, allRepoNames)
		matchingRepoNameSet := sets.String{}
		matchingRepoNameSet.Insert(matchingRepoNames...)

		for _, project := range projects {
			if matchingRepoNameSet.Has(project.Name) {
				matchingProjects = append(matchingProjects, project)
			}
		}
	} else {
		project, _, err := glc.Projects.GetProject(selector.ProjectID, &gitlab.GetProjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch gitlab projects: %w", err)
		}
		matchingProjects = append(matchingProjects, project)
	}
	return matchingProjects, nil
}

func (d *PackageDiscovery) getGitLabPackagesForProject(ctx context.Context, glc *gitlab.Client, project *gitlab.Project, selector gitopsv1alpha1.GitLabSelector) ([]DiscoveredPackage, error) {
	discoveredPackages := []DiscoveredPackage{}

	if isSelectorField(selector.Directory) {
		options := &gitlab.ListTreeOptions{
			Recursive: gitlab.Bool(true),
			// Ref:       gitlab.String(ref),
			// Path:      gitlab.String(path),
		}
		tree, _, err := glc.Repositories.ListTree(project.ID, options)
		if err != nil {
			return nil, err
		}

		allDirectories := []string{}
		for _, item := range tree {
			if item.Type == "tree" {
				// Directory
				allDirectories = append(allDirectories, item.Path)
			}
		}

		directories := filterByPattern(selector.Directory, allDirectories)
		for _, directory := range directories {
			thisDiscoveredPackage := DiscoveredPackage{
				Repo:          project.Name,
				Revision:      selector.Revision,
				Directory:     directory,
				GitLabProject: project,
			}
			discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
		}
	} else {
		thisDiscoveredPackage := DiscoveredPackage{
			Repo:          project.Name,
			Revision:      selector.Revision,
			Directory:     selector.Directory,
			GitLabProject: project,
		}
		discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
	}
	return discoveredPackages, nil
}
