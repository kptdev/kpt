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
	Name      string
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

	tree, _, err := gitClient.Git.GetTree(ctx, gitRepoSelector.Org, gitRepoSelector.Name, gitRepoSelector.Revision, true)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch tree from git: %w", err)
	}

	allPaths := []string{}
	for _, entry := range tree.Entries {
		if *entry.Type == "tree" {
			allPaths = append(allPaths, *entry.Path)
		}
	}

	packagesPaths := discoverPackagePaths(gitRepoSelector.PackagesPath, allPaths)

	discoveredPackages := []DiscoveredPackage{}

	for _, path := range packagesPaths {
		thisDiscoveredPackage := DiscoveredPackage{Org: gitRepoSelector.Org, Name: gitRepoSelector.Name, Revision: gitRepoSelector.Revision, Directory: path}
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

func discoverPackagePaths(pattern string, paths []string) []string {
	packagePaths := []string{}

	for _, path := range paths {
		if isMatch, _ := filepath.Match(pattern, path); isMatch {
			packagePaths = append(packagePaths, path)
		}
	}

	return packagePaths
}
