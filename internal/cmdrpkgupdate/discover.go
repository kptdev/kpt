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
	var prs []porchapi.PackageRevision
	var errs []string
	if len(args) == 0 || r.discover == downstream {
		prs = r.prs
	} else {
		for i := range args {
			pr := r.findPackageRevision(args[i])
			if pr == nil {
				errs = append(errs, fmt.Sprintf("could not find package revision %s", args[i]))
				continue
			}
			prs = append(prs, *pr)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors:\n  %s", strings.Join(errs, "\n  "))
	}

	repositories, err := r.getRepositories()
	if err != nil {
		return err
	}

	var upstreamUpdates [][]string
	downstreamUpdatesMap := make(map[string][]porchapi.PackageRevision) // map from the upstream package revision to a list of its downstream package revisions

	for _, pr := range prs {
		availableUpdates, upstreamName := r.availableUpdates(pr.Status.UpstreamLock, repositories)

		switch r.discover {
		case upstream:
			if len(availableUpdates) == 0 {
				upstreamUpdates = append(upstreamUpdates, []string{pr.Name, upstreamName, "No update available"})
			} else {
				var revisions []string
				for i := range availableUpdates {
					revisions = append(revisions, availableUpdates[i].Spec.Revision)
				}
				upstreamUpdates = append(upstreamUpdates, []string{pr.Name, upstreamName, strings.Join(revisions, ", ")})
			}
		case downstream:
			for _, update := range availableUpdates {
				key := update.Name + ":" + update.Spec.Revision
				downstreamUpdatesMap[key] = append(downstreamUpdatesMap[key], pr)
			}
		default:
			// this should never happen, because we validate in preRunE
			return fmt.Errorf("invalid argument %q for --discover", r.discover)
		}
	}

	switch r.discover {
	case upstream:
		return printupstreamUpdates(upstreamUpdates, cmd)
	case downstream:
		return printdownstreamUpdates(downstreamUpdatesMap, args, cmd)
	default:
		// this should never happen, because we validate in preRunE
		return fmt.Errorf("invalid argument %q for --discover", r.discover)
	}

}

func (r *runner) availableUpdates(upstreamLock *porchapi.UpstreamLock, repositories *configapi.RepositoryList) ([]porchapi.PackageRevision, string) {
	var availableUpdates []porchapi.PackageRevision
	var upstream string

	if upstreamLock == nil || upstreamLock.Git == nil {
		return nil, ""
	}
	// separate the revision number from the package name
	lastIndex := strings.LastIndex(upstreamLock.Git.Ref, "v")
	if lastIndex < 0 {
		return nil, ""
	}
	currentUpstreamRevision := upstreamLock.Git.Ref[lastIndex:]

	// upstream.git.ref could look like drafts/pkgname/version or pkgname/version
	upstreamPackageName := upstreamLock.Git.Ref[:lastIndex-1]
	upstreamPackageName = strings.TrimPrefix(upstreamPackageName, "drafts/")

	if !strings.HasSuffix(upstreamLock.Git.Repo, ".git") {
		upstreamLock.Git.Repo += ".git"
	}

	// find a repo that matches the upstreamLock
	var revisions []porchapi.PackageRevision
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
		switch cmp := semver.Compare(upstreamRevision.Spec.Revision, currentUpstreamRevision); {
		case cmp > 0: // upstreamRevision > currentUpstreamRevision
			availableUpdates = append(availableUpdates, upstreamRevision)
		case cmp == 0, cmp < 0: // upstreamRevision <= currentUpstreamRevision, do nothing
		}
	}

	return availableUpdates, upstream
}

// fetches all registered repositories
func (r *runner) getRepositories() (*configapi.RepositoryList, error) {
	repoList := configapi.RepositoryList{}
	err := r.client.List(r.ctx, &repoList, &client.ListOptions{})
	return &repoList, err
}

// fetches all package revision numbers for packages with the name upstreamPackageName from the repo
func (r *runner) getUpstreamRevisions(repo configapi.Repository, upstreamPackageName string) []porchapi.PackageRevision {
	var result []porchapi.PackageRevision
	for _, pkgRev := range r.prs {
		if pkgRev.Spec.Lifecycle != porchapi.PackageRevisionLifecyclePublished {
			// only consider published packages
			continue
		}
		if pkgRev.Spec.RepositoryName == repo.Name && pkgRev.Spec.PackageName == upstreamPackageName {
			result = append(result, pkgRev)
		}
	}
	return result
}

func printupstreamUpdates(upstreamUpdates [][]string, cmd *cobra.Command) error {
	w := printers.GetNewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "PACKAGE REVISION\tUPSTREAM REPOSITORY\tUPSTREAM UPDATES"); err != nil {
		return err
	}
	for _, pkgRev := range upstreamUpdates {
		if _, err := fmt.Fprintln(w, strings.Join(pkgRev, "\t")); err != nil {
			return err
		}
	}
	return w.Flush()
}

func printdownstreamUpdates(downstreamUpdatesMap map[string][]porchapi.PackageRevision, args []string, cmd *cobra.Command) error {
	var downstreamUpdates [][]string
	for upstreamPkgRev, downstreamPkgRevs := range downstreamUpdatesMap {
		split := strings.Split(upstreamPkgRev, ":")
		upstreamPkgRevName := split[0]
		upstreamPkgRevNum := split[1]
		for _, downstreamPkgRev := range downstreamPkgRevs {
			// figure out which upstream revision the downstream revision is based on
			lastIndex := strings.LastIndex(downstreamPkgRev.Status.UpstreamLock.Git.Ref, "v")
			if lastIndex < 0 {
				// this ref isn't formatted the way that porch expects
				continue
			}
			downstreamRev := downstreamPkgRev.Status.UpstreamLock.Git.Ref[lastIndex:]
			downstreamUpdates = append(downstreamUpdates, []string{upstreamPkgRevName, upstreamPkgRevNum, downstreamPkgRev.Name, downstreamRev})
		}
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())
	var pkgRevsToPrint [][]string
	if len(args) != 0 {
		for _, arg := range args {
			for _, pkgRev := range downstreamUpdates {
				// filter out irrelevant packages based on provided args
				if arg != pkgRev[0] {
					continue
				}
				pkgRevsToPrint = append(pkgRevsToPrint, pkgRev)
			}
		}
	} else {
		pkgRevsToPrint = downstreamUpdates
	}

	if len(pkgRevsToPrint) == 0 {
		if _, err := fmt.Fprintln(w, "All downstream packages are up to date."); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(w, "UPSTREAM PACKAGE\tUPSTREAM REVISION\tDOWNSTREAM PACKAGE\tDOWNSTREAM REVISION"); err != nil {
			return err
		}
		for _, pkgRev := range pkgRevsToPrint {
			if _, err := fmt.Fprintln(w, strings.Join(pkgRev, "\t")); err != nil {
				return err
			}
		}
	}

	return w.Flush()
}
