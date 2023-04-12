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

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/xanzy/go-gitlab"
	coreapi "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/kyaml/sets"
)

func (d *PackageDiscovery) NewGitLabClient(ctx context.Context, gitlabSource gitopsv1alpha1.GitLabSource) (*gitlab.Client, error) {

	// initialize a gitlab client
	secretName := gitlabSource.SecretRef.Name
	if secretName == "" {
		return nil, fmt.Errorf("gitlab secret reference is missing from the config")
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

	branch := selector.Branch
	if branch == "" {
		branch = project.DefaultBranch
	}
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
				Revision:      selector.Revision,
				Directory:     directory,
				GitLabProject: project,
				Branch:        branch,
			}
			discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
		}
	} else {
		thisDiscoveredPackage := DiscoveredPackage{
			Revision:      selector.Revision,
			Directory:     selector.Directory,
			GitLabProject: project,
			Branch:        branch,
		}
		discoveredPackages = append(discoveredPackages, thisDiscoveredPackage)
	}
	return discoveredPackages, nil
}
