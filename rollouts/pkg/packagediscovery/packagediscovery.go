// Copyright 2022 The kpt Authors
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
	"regexp"
	"strings"
	"sync"
	"time"

	gitopsv1alpha1 "github.com/GoogleContainerTools/kpt/rollouts/api/v1alpha1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v48/github"
	"github.com/xanzy/go-gitlab"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PackageDiscovery struct {
	client            client.Client
	namespace         string
	mutex             sync.Mutex
	cache             *Cache
	gitLabClientMaker GitLabClientMaker
	gitHubClientMaker GitHubClientMaker
}

// Maker interfaces to help with injecting our own clients during testing

// GitLabClientMaker knows how to make a GitLab Client.
type GitLabClientMaker interface {
	NewGitLabClient(ctx context.Context, config gitopsv1alpha1.GitLabSource) (*gitlab.Client, error)
}

// GitHubClientMaker knows how to make a GitHub Client.
type GitHubClientMaker interface {
	NewGitHubClient(ctx context.Context, config gitopsv1alpha1.GitHubSelector) (*github.Client, error)
}

// DiscoveredPackage represents a config package that will
// be rolled out.
type DiscoveredPackage struct {
	// User specified properties
	Directory string
	Revision  string
	Branch    string

	// Discovered properties of the project/repo

	// GitLabProject contains the info retrieved from GitLab
	GitLabProject *gitlab.Project
	// GithubRepo contains the info retrieved from GitHub
	GitHubRepo *github.Repository
	// OciRepo contains info retrieved from the OCI registry
	OciRepo *OCIRepo
}

type OCIRepo struct {
	Image string
}

// HTTPURL refers to the HTTP URL for the repository.
func (dp *DiscoveredPackage) HTTPURL() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.HTTPURLToRepo
	case dp.GitHubRepo != nil:
		return dp.GitHubRepo.GetCloneURL()
	}
	return ""
}

// SSHURL refers to the SSH(Git) URL for the repository.
func (dp *DiscoveredPackage) SSHURL() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.SSHURLToRepo
	case dp.GitHubRepo != nil:
		return dp.GitHubRepo.GetSSHURL()
	}
	return ""
}

// ID returns an identifier for the package.
// This is currently being used to generate the unique name
// of the RemoteSync object.
// TODO (droot): figure out a naming scheme for the package identity.
func (dp *DiscoveredPackage) ID() (id string) {
	switch {
	case dp.GitLabProject != nil:
		id = "gitlab-" + fmt.Sprintf("%d", dp.GitLabProject.ID)
	case dp.GitHubRepo != nil:
		id = "github-" + fmt.Sprintf("%d", dp.GitHubRepo.GetID())
	default:
		return ""
	}
	if dp.Directory == "" || dp.Directory == "." || dp.Directory == "/" {
		return id
	}
	return id + fmt.Sprintf("-%s", dp.Directory)
}

func (dp *DiscoveredPackage) String() string {
	switch {
	case dp.GitLabProject != nil:
		return dp.GitLabProject.Name
	case dp.GitHubRepo != nil:
		return *dp.GitHubRepo.Name
	case dp.OciRepo != nil:
		return dp.OciRepo.Image
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
	case gitopsv1alpha1.OCI:
		discoveredPackages, err = d.getOCIPackages(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("unable to fetch OCI packages: %w", err)
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

// ToStr is convenient method to pretty print set of packages.
func ToStr(packages []DiscoveredPackage) string {
	pkgNames := []string{}
	for _, pkg := range packages {
		pkgNames = append(pkgNames, pkg.String())
	}
	return strings.Join(pkgNames, ",")
}
