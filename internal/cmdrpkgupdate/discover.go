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

package cmdrpkgupdate

import (
	"fmt"
	"strings"

	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *runner) discoverUpdates(cmd *cobra.Command, args []string) error {
	var errs []string
	var updates [][]string

	var prs []porchapi.PackageRevision
	if len(args) == 0 {
		prs = r.prs
	} else {
		for i := range args {
			pr := r.findPackageRevision(args[i])
			if pr == nil {
				return fmt.Errorf("could not find package revision %s", args[i])
			}
			prs = append(prs, *pr)
		}
	}

	for _, pr := range prs {
		availableUpdates, upstreamName, err := r.availableUpdates(pr.Status.UpstreamLock)
		if err != nil {
			return err
		}
		if len(availableUpdates) == 0 {
			updates = append(updates, []string{pr.Name, upstreamName, "No update available"})
		} else {
			updates = append(updates, []string{pr.Name, upstreamName, strings.Join(availableUpdates, ", ")})
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors:\n  %s", strings.Join(errs, "\n  "))
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "PACKAGE REVISION\tUPSTREAM REPOSITORY\tUPSTREAM UPDATES"); err != nil {
		return err
	}
	for _, pkgRev := range updates {
		if _, err := fmt.Fprintln(w, strings.Join(pkgRev, "\t")); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func (r *runner) availableUpdates(upstreamLock *porchapi.UpstreamLock) ([]string, string, error) {
	var availableUpdates []string
	var upstream string

	if upstreamLock == nil || upstreamLock.Git == nil {
		return nil, "", nil
	}
	// separate the revision number from the package name
	lastIndex := strings.LastIndex(upstreamLock.Git.Ref, "v")
	if lastIndex < 0 {
		return nil, "", nil
	}
	currentUpstreamRevision := upstreamLock.Git.Ref[lastIndex:]

	// upstream.git.ref could look like drafts/pkgname/version or pkgname/version
	upstreamPackageName := upstreamLock.Git.Ref[:lastIndex-1]
	upstreamPackageName = strings.TrimPrefix(upstreamPackageName, "drafts/")

	if !strings.HasSuffix(upstreamLock.Git.Repo, ".git") {
		upstreamLock.Git.Repo += ".git"
	}

	repositories, err := r.getRepositories()
	if err != nil {
		return nil, "", err
	}

	// find a repo that matches the upstreamLock
	var revisions []string
	for _, repo := range repositories.Items {
		if repo.Spec.Type != configapi.RepositoryTypeGit {
			// we are not currently supporting non-git repos for updates
			continue
		}
		if !strings.HasSuffix(repo.Spec.Git.Repo, ".git") {
			repo.Spec.Git.Repo += ".git"
		}
		if upstreamLock.Git.Repo == repo.Spec.Git.Repo {
			upstream = repo.Name
			revisions = r.getUpstreamRevisions(repo, upstreamPackageName)
		}
	}

	for _, upstreamRevision := range revisions {
		switch cmp := semver.Compare(upstreamRevision, currentUpstreamRevision); {
		case cmp > 0: // upstreamRevision > currentUpstreamRevision
			availableUpdates = append(availableUpdates, upstreamRevision)
		case cmp == 0, cmp < 0: // upstreamRevision <= currentUpstreamRevision, do nothing
		}
	}

	return availableUpdates, upstream, nil
}

// fetches all registered repositories
func (r *runner) getRepositories() (*configapi.RepositoryList, error) {
	repoList := configapi.RepositoryList{}
	err := r.client.List(r.ctx, &repoList, &client.ListOptions{})
	return &repoList, err
}

// fetches all package revision numbers for packages with the name upstreamPackageName from the repo
func (r *runner) getUpstreamRevisions(repo configapi.Repository, upstreamPackageName string) []string {
	var result []string
	for _, pkgRev := range r.prs {
		if pkgRev.Spec.Lifecycle != porchapi.PackageRevisionLifecyclePublished {
			// only consider published packages
			continue
		}
		if pkgRev.Spec.RepositoryName != repo.Name ||
			pkgRev.Spec.PackageName != upstreamPackageName {
			continue
		}
		result = append(result, pkgRev.Spec.Revision)
	}
	return result
}
