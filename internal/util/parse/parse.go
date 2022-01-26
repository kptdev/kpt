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

package parse

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	kpterrors "github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-containerregistry/pkg/name"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

type Options struct {
	SetGit func(git *kptfilev1.Git) error
	SetOci func(oci *kptfilev1.Oci) error
}

func ParseArgs(ctx context.Context, args []string, opts Options) (string, error) {
	const op kpterrors.Op = "parse.ParseArgs"

	tGit, errGit := GitParseArgs(ctx, args)
	if errGit == nil {
		if opts.SetGit == nil {
			return "", kpterrors.E(op, fmt.Errorf("git locations not supported: %v", errGit))
		}
		if err := opts.SetGit(&tGit.Git); err != nil {
			return "", err
		}
		return tGit.Destination, nil
	}

	tOci, errOci := OciParseArgs(ctx, args)
	if errOci == nil {
		if opts.SetOci == nil {
			return "", kpterrors.E(op, fmt.Errorf("oci locations not supported: %v", errOci))
		}
		if err := opts.SetOci(&tOci.Oci); err != nil {
			return "", err
		}
		return tOci.Destination, nil
	}

	// TODO(oci-support) combining error messages like this is suboptimal in several ways
	return "", kpterrors.E(op, fmt.Errorf("%v %v", errGit, errOci))
}

type OciTarget struct {
	kptfilev1.Oci
	Destination string
}

func OciParseArgs(ctx context.Context, args []string) (OciTarget, error) {
	oci := OciTarget{}
	if args[0] == "-" {
		return oci, nil
	}

	// The prefix must occur, and must not have other characters before it
	arg0parts := strings.SplitN(args[0], "oci://", 2)
	if len(arg0parts) != 2 || len(arg0parts[0]) != 0 {
		return oci, errors.Errorf("ambiguous image:tag specify 'oci://' before argument: %s", args[0])
	}

	return targetFromImageReference(arg0parts[1], args[1])
}

func targetFromImageReference(image, dest string) (OciTarget, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return OciTarget{}, err
	}

	registry := ref.Context().RegistryStr()
	repository := ref.Context().RepositoryStr()
	destination, err := getDest(dest, registry, repository)
	if err != nil {
		return OciTarget{}, err
	}

	directory := ""
	parts := strings.SplitN(ref.Context().Name(), "//", 2)
	if len(parts) == 2 {
		directory = "/" + parts[1]
		repo, err := name.NewRepository(parts[0])
		if err != nil {
			return OciTarget{}, err
		}

		switch r := ref.(type) {
		case name.Tag:
			ref = repo.Tag(r.TagStr())
		case name.Digest:
			ref = repo.Tag(r.DigestStr())
		}
	}

	return OciTarget{
		Oci: kptfilev1.Oci{
			Image:     ref.Name(),
			Directory: directory,
		},
		Destination: destination,
	}, nil
}

type GitTarget struct {
	kptfilev1.Git
	Destination string
}

func GitParseArgs(ctx context.Context, args []string) (GitTarget, error) {
	g := GitTarget{}
	if args[0] == "-" {
		return g, nil
	}

	// Simple parsing if contains .git
	if strings.Contains(args[0], ".git") {
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

	g.Ref = version
	g.Directory = path.Clean(remoteDir)
	g.Repo = repo

	if len(args) >= 2 {
		destination, err := getDest(args[1], repo, remoteDir)
		if err != nil {
			return g, err
		}
		g.Destination = filepath.Clean(destination)
	}
	return g, nil
}

// targetFromPkgURL parses a pkg url and destination into kptfile git info and local destination Target
func targetFromPkgURL(ctx context.Context, pkgURL, dest string) (GitTarget, error) {
	g := GitTarget{}
	var repo, dir, version string
	parts := strings.Split(pkgURL, ".git")
	repo = strings.TrimSuffix(parts[0], "/")
	switch {
	case len(parts) == 1:
		// do nothing
	case strings.Contains(parts[1], "@"):
		parts := strings.Split(parts[1], "@")
		version = strings.TrimSuffix(parts[1], "/")
		dir = parts[0]
	default:
		dir = parts[1]
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
	if dir == "" {
		dir = "/"
	}
	destination, err := getDest(dest, repo, dir)
	if err != nil {
		return g, err
	}
	g.Ref = version
	g.Directory = path.Clean(dir)
	g.Repo = repo
	g.Destination = filepath.Clean(destination)
	return g, nil
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
	// v is "" for commands that do not require an output path
	if v == "" {
		return "", nil
	}

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
