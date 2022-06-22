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

package cmdrpkgfindupdates

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/util/porch"
	porchapi "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	command = "cmdrpkgfindupdates"
)

func NewCommand(ctx context.Context, rcg *genericclioptions.ConfigFlags) *cobra.Command {
	return newRunner(ctx, rcg).Command
}

func newRunner(ctx context.Context, rcg *genericclioptions.ConfigFlags) *runner {
	r := &runner{
		ctx:    ctx,
		cfg:    rcg,
		client: nil,
	}

	c := &cobra.Command{
		Use: "find-updates PACKAGE_REVISION",
		//Short:   rpkgdocs.FindUpdatesShort,
		//Long:    rpkgdocs.FindUpdatesShort + "\n" + rpkgdocs.FindUpdatesLong,
		//Example: rpkgdocs.FindUpdatesExamples,
		PreRunE: r.preRunE,
		RunE:    r.runE,
		Hidden:  porch.HidePorchCommands,
	}
	r.Command = c
	return r
}

type runner struct {
	ctx     context.Context
	cfg     *genericclioptions.ConfigFlags
	client  client.Client
	Command *cobra.Command

	// there are multiple places where we need access to all package revisions, so
	// we store it in the runner
	prs []porchapi.PackageRevision

	// Flags
}

func (r *runner) preRunE(_ *cobra.Command, args []string) error {
	const op errors.Op = command + ".preRunE"

	c, err := porch.CreateClient(r.cfg)
	if err != nil {
		return errors.E(op, err)
	}
	r.client = c

	if len(args) == 0 {
		// if no arguments provided, show results for all package revisions
		packageRevisionList := porchapi.PackageRevisionList{}
		if err := r.client.List(r.ctx, &packageRevisionList, &client.ListOptions{}); err != nil {
			return errors.E(op, err)
		}
		r.prs = packageRevisionList.Items
	} else {
		var packageRevisions []porchapi.PackageRevision
		for _, p := range args {
			packageRevision, err := r.getPackageRevision(p)
			if err != nil {
				return errors.E(op, err)
			}
			packageRevisions = append(packageRevisions, packageRevision)
		}
		r.prs = packageRevisions
	}

	return nil
}

func (r *runner) runE(cmd *cobra.Command, args []string) error {
	const op errors.Op = command + ".runE"
	var errs []string
	var updates [][]string

	for _, pr := range r.prs {
		availableUpdates, upstreamName, err := r.availableUpdates(pr.Status.UpstreamLock)
		if err != nil {
			return errors.E(op, err)
		}
		if len(availableUpdates) == 0 {
			updates = append(updates, []string{pr.Name, upstreamName, "No update available"})
		} else {
			updates = append(updates, []string{pr.Name, upstreamName, strings.Join(availableUpdates, ", ")})
		}
	}

	if len(errs) > 0 {
		return errors.E(op, fmt.Errorf("errors:\n  %s", strings.Join(errs, "\n  ")))
	}

	w := printers.GetNewTabWriter(cmd.OutOrStdout())
	if _, err := fmt.Fprintln(w, "PACKAGE REVISION\tUPSTREAM REPOSITORY\tUPSTREAM UPDATES"); err != nil {
		return errors.E(op, err)
	}
	for _, pkgRev := range updates {
		if _, err := fmt.Fprintln(w, strings.Join(pkgRev, "\t")); err != nil {
			return errors.E(op, err)
		}
	}
	if err := w.Flush(); err != nil {
		return errors.E(op, err)
	}

	return nil
}

func (r *runner) getPackageRevision(p string) (porchapi.PackageRevision, error) {
	fmt.Println("getPackageRevision")

	pr := porchapi.PackageRevision{}
	err := r.client.Get(r.ctx, client.ObjectKey{
		Namespace: *r.cfg.Namespace,
		Name:      p,
	}, &pr)

	return pr, err
}

func (r *runner) availableUpdates(upstreamLock *porchapi.UpstreamLock) ([]string, string, error) {
	var availableUpdates []string
	var upstream string

	if upstreamLock == nil || upstreamLock.Git == nil {
		return nil, "", nil
	}
	// separate the revision number from the package name
	lastIndex := strings.LastIndex(upstreamLock.Git.Ref, "v")
	currentUpstreamRevision := upstreamLock.Git.Ref[lastIndex:]
	upstreamPackageName := upstreamLock.Git.Ref[:lastIndex-1]

	if !strings.HasSuffix(upstreamLock.Git.Repo, ".git") {
		upstreamLock.Git.Repo += ".git"
	}

	repositories, err := r.getRepositories()
	if err != nil {
		return nil, "", err
	}

	// find a repo that matches the upstreamLock
	var revisions []string
	fmt.Println("iterating over repositories")
	for _, repo := range repositories.Items {
		fmt.Println(repo.Spec.Git.Repo, upstreamLock.Git.Repo)
		if repo.Spec.Type != configapi.RepositoryTypeGit {
			// we are not currently supporting non-git repos for updates
			continue
		}
		if upstreamLock.Git.Repo == repo.Spec.Git.Repo {
			upstream = repo.Name
			revisions, err = r.getUpstreamRevisions(repo, upstreamPackageName)
			if err != nil {
				return nil, "", err
			}
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
	fmt.Println("getRepositories")
	repoList := configapi.RepositoryList{}
	err := r.client.List(r.ctx, &repoList, &client.ListOptions{})
	for _, k := range repoList.Items {
		fmt.Println(k.Name)
	}

	return &repoList, err
}

// fetches all package revision numbers for packages with the name upstreamPackageName from the repo
func (r *runner) getUpstreamRevisions(repo configapi.Repository, upstreamPackageName string) ([]string, error) {
	fmt.Println("getUpstreamRevisions")
	fmt.Println(repo.Name)

	var result []string
	for _, pkgRev := range r.prs {
		if pkgRev.Spec.RepositoryName != repo.Name ||
			pkgRev.Spec.PackageName != upstreamPackageName {
			continue
		}
		result = append(result, pkgRev.Spec.Revision)
	}
	return result, nil
}
