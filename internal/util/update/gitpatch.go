// Copyright 2019 Google LLC
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

package update

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

// Updater updates a package to a new upstream version.
//
// If the package at pkgPath differs from the upstream ref it was fetch from, then Update will
// attempt to create a patch from the upstream source version and upstream update version.
type GitPatchUpdater struct {
	UpdateOptions

	// patch is a patch which can be applied with 'git am'
	patch string

	// toCommit is resolved commit for toRef
	toCommit string

	// gitRunner is used to run git commands
	gitRunner *gitutil.GitRunner

	// packageRef is the RemoteDirectory/ToRef -- for sub directory versioning
	packageRef string
}

func (u GitPatchUpdater) Update(options UpdateOptions) error {
	u.UpdateOptions = options
	u.packageRef = path.Join(strings.TrimLeft(u.KptFile.Upstream.Git.Directory, "/"),
		u.ToRef)
	if err := u.calculatePatch(); err != nil {
		return err
	}

	// write the patch to a file instead of applying it
	if options.DryRun {
		_, err := io.WriteString(os.Stderr, fmt.Sprintf(
			"patch can be applied with 'git am -3 --directory %s'\n", options.PackagePath))
		if err != nil {
			return err
		}
		_, err = io.WriteString(options.Output, u.patch)
		return err
	}

	// apply the patch to pkg
	return u.patchLocalPackage()
}

// calculatePatch runs a series of git commands to calculate a patch which can be used
// to update the local package
func (u *GitPatchUpdater) calculatePatch() error {
	var err error

	optional := []string{u.ToRef}
	if u.packageRef != u.ToRef {
		optional = append(optional, u.ToRef)
	}
	if u.gitRunner, err = gitutil.NewUpstreamGitRunner(
		u.KptFile.Upstream.Git.Repo, u.KptFile.Upstream.Git.Directory,
		[]string{u.UpdateOptions.KptFile.Upstream.Git.Commit},
		optional,
	); err != nil {
		return err
	}
	u.gitRunner.Verbose = u.Verbose

	// record the destination commit to upgrade from in the future
	if err := u.destinationRefToCommitSha(); err != nil {
		return err
	}

	// reset to the point in time we are updating from
	if err := u.hardResetSourceFiles(); err != nil {
		return err
	}

	// add a commit with the files we want to update to
	if err := u.commitTargetFiles(); err != nil {
		return err
	}

	// generate the patch between from and to
	return u.formatPatch()
}

const alphaGitPatchRemote = "kpt-update-alpha-git-patch"

// patchLocalPackage will run 'git am' to patch the local package.
func (u *GitPatchUpdater) patchLocalPackage() error {
	g := gitutil.NewLocalGitRunner(u.UpdateOptions.PackagePath)

	// add the cached update as an upstream so git can figure out how to do the
	// 3-way merge when it looks for the commits in the patch file.
	fmt.Fprintf(os.Stderr,
		"fetching upstream updates locally staged at '%s'\n", u.gitRunner.RepoDir)
	// TODO(pwittrock): consider fetching directly without adding using git fetch <path>
	//                  and determine if there are any benefits in doing so over this approach.
	if err := g.Run(
		"remote", "add", alphaGitPatchRemote, u.gitRunner.RepoDir); err != nil {
		return errors.Errorf("update failed: failure running git remote '%v': %s %s",
			err, g.Stderr.String(), g.Stdout.String())
	}
	defer func() {
		// delete the remote when we are done
		err := g.Run("remote", "remove", alphaGitPatchRemote)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cleanup remote failed: %v\n", err)
		}
	}()
	defaultRef, err := gitutil.DefaultRef(u.UpdateOptions.ToRepo)
	if err != nil {
		return err
	}
	if err := g.Run("fetch", alphaGitPatchRemote, defaultRef); err != nil {
		return errors.Errorf("update failed: failure running git fetch '%v': %s %s",
			err, g.Stderr.String(), g.Stdout.String())
	}

	// run `git am` to apply the patch
	fmt.Fprintf(os.Stderr,
		"applying upstream updates using `git am -3 --directory %s`\n", u.PackagePath)
	g.Stdin = &bytes.Buffer{}
	if _, err := g.Stdin.WriteString(u.patch); err != nil {
		return err
	}
	if err := g.Run("am", "-3", "--directory", u.PackagePath); err != nil {
		return errors.Errorf("update failed: failure running git am: '%v': %s %s",
			err, g.Stderr.String(), g.Stdout.String())
	}
	return nil
}

// hardResetSourceFiles hard resets the repo to the commit we are updating from.
// it also writes the Kptfile from the local package so that the patch correctly updates it
// to include the new commit, ref, repo post update.
func (u *GitPatchUpdater) hardResetSourceFiles() error {
	// reset hard to where we are updating from
	if err := u.gitRunner.Run(
		"reset", "--hard", u.UpdateOptions.KptFile.Upstream.Git.Commit); err != nil {
		return errors.Errorf("update failed: failure running git reset --hard: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}

	pf, err := kptfileutil.ReadFile(u.gitRunner.Dir)
	if err != nil {
		// no upstream Kptfile, use our local copy -- use the local Kptfile value.
		pf = u.UpdateOptions.KptFile
	} else {
		// found upstream Kptfile, use the upstream copy, but set the `upstream` field
		// since it is owned locally
		pf.Upstream = u.UpdateOptions.KptFile.Upstream
		// also keep the local OpenAPI which may have been modified.
		err = pf.MergeOpenAPI(u.UpdateOptions.KptFile, u.UpdateOptions.KptFile)
		if err != nil {
			return err
		}
	}

	// write the Kptfile so the patch sees the updates to it.  use the version we read
	// locally so there aren't merge conflicts if the remote was changed.
	if err := kptfileutil.WriteFile(u.gitRunner.Dir, pf); err != nil {
		return errors.Errorf("update failed: unable to write Kptfile: '%v'", err)
	}
	if err := u.gitRunner.Run("add", "Kptfile"); err != nil {
		return errors.Errorf("update failed: unable to add Kptfile: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	if err := u.gitRunner.Run("commit", "-m", "write from KptFile"); err != nil {
		return errors.Errorf("update failed: unable to commit source: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	return nil
}

// commitTargetFiles checks out the files we are updating to, and updates the Kptfile with the
// values for
func (u *GitPatchUpdater) commitTargetFiles() error {
	// checkout the files we want to update to
	if err := u.gitRunner.Run("checkout", u.toCommit, "./"); err != nil {
		return errors.Errorf("update failed: unable to checkout update target '%s': '%v': %s %s",
			u.toCommit, err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}

	// found a remote package Kptfile -- take this one over any default that we generated
	updatedKptfile, err := kptfileutil.ReadFile(u.gitRunner.Dir)
	if err != nil {
		updatedKptfile, err = kptfileutil.ReadFile(u.PackagePath)
		if err != nil {
			return errors.Errorf("update failed: unable to read Kptfile: %v", err)
		}
	}

	// write the updated Kptfile so changes to it are included in the patch
	updatedKptfile.Upstream.Git.Commit = u.toCommit           // set the commit we are updating to
	updatedKptfile.Upstream.Git.Ref = u.UpdateOptions.ToRef   // set the ref we are updating to
	updatedKptfile.Upstream.Git.Repo = u.UpdateOptions.ToRepo // set the repo we are using for the update
	if err := kptfileutil.WriteFile(u.gitRunner.Dir, updatedKptfile); err != nil {
		return errors.Errorf("update failed: unable to write Kptfile: '%v'", err)
	}

	// add and commit the files and Kptfile so we can create a patch for them
	if err := u.gitRunner.Run("add", "."); err != nil {
		return errors.Errorf("update failed: unable to add update target: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	var msg string
	if u.SimpleMessage {
		msg = fmt.Sprintf("update from '%s' to '%s'",
			u.UpdateOptions.KptFile.Upstream.Git.Ref, // ref we are updating from
			u.UpdateOptions.ToRef,                    // ref we are updating to
		)
	} else {
		msg = fmt.Sprintf("update '%s' (%s) from '%s' (%s) to '%s' (%s)",
			u.UpdateOptions.KptFile.Name,             // name of the package
			u.UpdateOptions.ToRepo,                   // repo used for the update
			u.UpdateOptions.KptFile.Upstream.Git.Ref, // ref we are updating from
			u.UpdateOptions.KptFile.Upstream.Git.Commit,
			u.UpdateOptions.ToRef, // ref we are updating to
			u.toCommit,
		)
	}

	if err := u.gitRunner.Run("diff", "--quiet", "HEAD"); err == nil {
		return errors.Errorf("no updates")
	}

	if err := u.gitRunner.Run("commit", "-m", msg); err != nil {
		return errors.Errorf("update failed: unable to commit update target: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	return nil
}

// formatPatch generates a patch for the most recent commit and records it on p.patch.
func (u *GitPatchUpdater) formatPatch() error {
	if err := u.gitRunner.Run(
		"format-patch", "--stdout", "-n", "HEAD^", "--relative", "./"); err != nil {
		return errors.Errorf("update failed: unable to create patch: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	u.patch = u.gitRunner.Stdout.String()
	return nil
}

// destinationRefToCommitSha resolves the destination ref to a commit and records it
// on p.toCommit
func (u *GitPatchUpdater) destinationRefToCommitSha() error {
	var err error

	// first check if there is a tag for the specific subdirectory for per-dir versioning
	if err = u.gitRunner.Run("reset", "--hard", u.packageRef); err != nil {
		// this works for tags
		if err = u.gitRunner.Run("reset", "--hard", u.ToRef); err != nil {
			// this works for branches
			if err = u.gitRunner.Run("reset", "--hard", "origin/"+u.ToRef); err != nil {
				return errors.Errorf("update failed: unable to reset to update target: '%v': %s %s",
					err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
			}
		}
	}
	if err := u.gitRunner.Run("rev-parse", "--verify", "HEAD"); err != nil {
		return errors.Errorf("update failed: unable to parse update target commit: '%v': %s %s",
			err, u.gitRunner.Stderr.String(), u.gitRunner.Stdout.String())
	}
	u.toCommit = strings.TrimSpace(u.gitRunner.Stdout.String())
	return nil
}
