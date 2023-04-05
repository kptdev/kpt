// Copyright 2019 The kpt Authors
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

package parse

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

const gitSuffixRegexp = "\\.git($|/)"

type Target struct {
	kptfilev1.Git
	Destination string
}

func GitParseArgs(ctx context.Context, args []string) (Target, error) {
	g := Target{}
	if args[0] == "-" {
		return g, nil
	}

	// Simple parsing if contains .git{$|/)
	if HasGitSuffix(args[0]) {
		return targetFromPkgURL(ctx, args[0], args[1])
	}

	// GitHub parsing if contains github.com
	if strings.Contains(args[0], "github.com") {
		ghPkgURL, err := pkgURLFromGHURL(ctx, args[0], getRepoBranches)
		if err != nil {
			return g, err
		}
		return targetFromPkgURL(ctx, ghPkgURL, args[1])
	}

	uri, version, err := getURIAndVersion(args[0])
	if err != nil {
		return g, err
	}
	repo, remoteDir, err := getRepoAndPkg(uri)
	if err != nil {
		return g, err
	}
	if version == "" {
		gur, err := gitutil.NewGitUpstreamRepo(ctx, repo)
		if err != nil {
			return g, err
		}
		defaultRef, err := gur.GetDefaultBranch(ctx)
		if err != nil {
			return g, err
		}
		version = defaultRef
	}

	destination, err := getDest(args[1], repo, remoteDir)
	if err != nil {
		return g, err
	}
	g.Ref = version
	g.Directory = path.Clean(remoteDir)
	g.Repo = repo
	g.Destination = filepath.Clean(destination)
	return g, nil
}

// targetFromPkgURL parses a pkg url and destination into kptfile git info and local destination Target
func targetFromPkgURL(ctx context.Context, pkgURL string, dest string) (Target, error) {
	g := Target{}
	repo, dir, ref, err := URL(pkgURL)
	if err != nil {
		return g, err
	}
	if dir == "" {
		dir = "/"
	}
	if ref == "" {
		gur, err := gitutil.NewGitUpstreamRepo(ctx, repo)
		if err != nil {
			return g, err
		}
		defaultRef, err := gur.GetDefaultBranch(ctx)
		if err != nil {
			return g, err
		}
		ref = defaultRef
	}
	destination, err := getDest(dest, repo, dir)
	if err != nil {
		return g, err
	}
	g.Ref = ref
	g.Directory = path.Clean(dir)
	g.Repo = repo
	g.Destination = filepath.Clean(destination)
	return g, nil
}

// URL parses a pkg url (must contain ".git") and returns the repo, directory, and version
func URL(pkgURL string) (repo string, dir string, ref string, err error) {
	parts := regexp.MustCompile(gitSuffixRegexp).Split(pkgURL, 2)
	index := strings.Index(pkgURL, parts[0])
	repo = strings.Join([]string{pkgURL[:index], parts[0]}, "")
	switch {
	case len(parts) == 1 || parts[1] == "":
		// do nothing
	case strings.Contains(parts[1], "@"):
		parts := strings.Split(parts[1], "@")
		ref = strings.TrimSuffix(parts[1], "/")
		dir = string(filepath.Separator) + parts[0]
	default:
		dir = string(filepath.Separator) + parts[1]
	}
	return repo, dir, ref, nil
}

// pkgURLFromGHURL converts a GitHub URL into a well formed pkg url
// by adding a .git suffix after repo URI and version info if available
func pkgURLFromGHURL(ctx context.Context, v string, findRepoBranches func(context.Context, string) ([]string, error)) (string, error) {
	v = strings.TrimSuffix(v, "/")
	// url should have scheme and host separated by ://
	parts := strings.SplitN(v, "://", 2)
	if len(parts) != 2 {
		return "", errors.Errorf("invalid GitHub url: %s", v)
	}
	// host should be github.com
	if !strings.HasPrefix(parts[1], "github.com") {
		return "", errors.Errorf("invalid GitHub url: %s", v)
	}

	ghRepoParts := strings.Split(parts[1], "/")
	// expect at least github.com/owner/repo
	if len(ghRepoParts) < 3 {
		return "", errors.Errorf("invalid GitHub pkg url: %s", v)
	}
	// url of form github.com/owner/repo
	if len(ghRepoParts) == 3 {
		repoWithPath := path.Join(ghRepoParts...)
		// return scheme://github.com/owner/repo.git
		return parts[0] + "://" + path.Join(repoWithPath) + ".git", nil
	}

	// url of form github.com/owner/repo/tree/ref/<path>
	if ghRepoParts[3] == "tree" && len(ghRepoParts) > 4 {
		repo := parts[0] + "://" + path.Join(ghRepoParts[:3]...)
		version := ghRepoParts[4]
		dir := path.Join(ghRepoParts[5:]...)
		// For an input like github.com/owner/repo/tree/feature/foo-feat where feature/foo-feat is the branch name
		// we will extract version as feature which is invalid.
		// To identify potential mismatch, we find all branches in the upstream repo
		// and check for potential matches, returning an error if any matched.
		branches, err := findRepoBranches(ctx, repo)
		if err != nil {
			return "", err
		}
		if isAmbiguousBranch(version, branches) {
			return "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument: %s", v)
		}

		if dir != "" {
			// return scheme://github.com/owner/repo.git/path@ref
			return fmt.Sprintf("%s.git/%s@%s", repo, dir, version), nil
		}
		// return scheme://github.com/owner/repo.git@ref
		return fmt.Sprintf("%s.git@%s", repo, version), nil
	}
	// if no tree, version info is unavailable in url
	// url of form github.com/owner/repo/<path>
	repo := fmt.Sprintf("%s://%s", parts[0], path.Join(ghRepoParts[:3]...))
	dir := path.Join(ghRepoParts[3:]...)
	// return scheme://github.com/owner/repo.git/path
	return repo + path.Join(".git", dir), nil
}

// getRepoBranches returns a slice of branches in upstream repo
func getRepoBranches(ctx context.Context, repo string) ([]string, error) {
	gur, err := gitutil.NewGitUpstreamRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	branches := make([]string, 0, len(gur.Heads))
	for head := range gur.Heads {
		branches = append(branches, head)
	}
	return branches, nil
}

// isAmbiguousBranch checks if a given branch name is similar to other branch names.
// If a branch with an appended slash matches other branches, then it is ambiguous.
func isAmbiguousBranch(branch string, branches []string) bool {
	branch += "/"
	for _, b := range branches {
		if strings.Contains(b, branch) {
			return true
		}
	}
	return false
}

// getURIAndVersion parses the repo+pkgURI and the version from v
func getURIAndVersion(v string) (string, string, error) {
	if strings.Count(v, "://") > 1 {
		return "", "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument")
	}
	if strings.Count(v, "@") > 2 {
		return "", "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument")
	}
	pkgURI := strings.SplitN(v, "@", 2)
	if len(pkgURI) == 1 {
		return pkgURI[0], "", nil
	}
	return pkgURI[0], pkgURI[1], nil
}

// getRepoAndPkg parses the repository uri and the package subdirectory from v
func getRepoAndPkg(v string) (string, string, error) {
	parts := strings.SplitN(v, "://", 2)
	if len(parts) != 2 {
		return "", "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument")
	}

	if strings.Count(v, ".git/") != 1 && !strings.HasSuffix(v, ".git") {
		return "", "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument")
	}

	if strings.HasSuffix(v, ".git") || strings.HasSuffix(v, ".git/") {
		v = strings.TrimSuffix(v, "/")
		v = strings.TrimSuffix(v, ".git")
		return v, "/", nil
	}

	repoAndPkg := strings.SplitN(v, ".git/", 2)
	return repoAndPkg[0], repoAndPkg[1], nil
}

func getDest(v, repo, subdir string) (string, error) {
	v = filepath.Clean(v)

	f, err := os.Stat(v)
	if os.IsNotExist(err) {
		parent := filepath.Dir(v)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			// error -- fetch to directory where parent does not exist
			return "", errors.Errorf("parent directory %q does not exist", parent)
		}
		// fetch to a specific directory -- don't default the name
		return v, nil
	}

	if !f.IsDir() {
		return "", errors.Errorf("LOCAL_PKG_DEST must be a directory")
	}

	// LOCATION EXISTS
	// default the location to a new subdirectory matching the pkg URI base
	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")
	v = filepath.Join(v, path.Base(path.Join(path.Clean(repo), path.Clean(subdir))))

	// make sure the destination directory does not yet exist yet
	if _, err := os.Stat(v); !os.IsNotExist(err) {
		return "", errors.Errorf("destination directory %q already exists", v)
	}
	return v, nil
}

// HasGitSuffix returns true if the provided pkgURL is a git repo containing the ".git" suffix
func HasGitSuffix(pkgURL string) bool {
	matched, err := regexp.Match(gitSuffixRegexp, []byte(pkgURL))
	return matched && err == nil
}
