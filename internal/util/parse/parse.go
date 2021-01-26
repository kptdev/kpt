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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/gitutil"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/kyaml/errors"
)

type Target struct {
	kptfile.Git
	Destination string
}

func GitParseArgs(args []string) (Target, error) {
	g := Target{}
	if args[0] == "-" {
		return g, nil
	}

	// Simple parsing if contains .git
	if strings.Contains(args[0], ".git") {
		var repo, dir, version string
		parts := strings.Split(args[0], ".git")
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
			defaultRef, err := gitutil.DefaultRef(repo)
			if err != nil {
				return g, err
			}
			version = defaultRef
		}
		if dir == "" {
			dir = "/"
		}
		destination, err := getDest(args[1], repo, dir)
		if err != nil {
			return g, err
		}
		g.Ref = version
		g.Directory = path.Clean(dir)
		g.Repo = repo
		g.Destination = filepath.Clean(destination)
		return g, nil
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
		defaultRef, err := gitutil.DefaultRef(repo)
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

	if strings.HasPrefix(parts[1], "github.com") {
		repoSubdir := append(strings.Split(parts[1], "/"), "/")
		if len(repoSubdir) < 4 {
			return "", "", errors.Errorf("ambiguous repo/dir@version specify '.git' in argument")
		}
		repo := parts[0] + "://" + path.Join(repoSubdir[:3]...)
		dir := path.Join(repoSubdir[3:]...)
		return repo, dir, nil
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
