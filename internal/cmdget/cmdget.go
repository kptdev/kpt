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

// Package cmdget contains the get command
package cmdget

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	docs "github.com/GoogleContainerTools/kpt/internal/docs/generated/commands"
	"github.com/GoogleContainerTools/kpt/internal/util/cmdutil"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/get/getioreader"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

// NewRunner returns a command runner
func NewRunner(parent string) *Runner {
	r := &Runner{}
	c := &cobra.Command{
		Use:        "get REPO_URI[.git]/PKG_PATH[@VERSION] LOCAL_DEST_DIRECTORY",
		Args:       cobra.ExactArgs(2),
		Short:      docs.GetShort,
		Long:       docs.GetLong,
		Example:    docs.GetExamples,
		RunE:       r.runE,
		PreRunE:    r.preRunE,
		SuggestFor: []string{"clone", "cp", "fetch"},
	}
	cmdutil.FixDocs("kpt", parent, c)
	r.Command = c
	c.Flags().StringVar(&r.FilenamePattern, "pattern", filters.DefaultFilenamePattern,
		`Pattern to use for writing files.  
May contain the following formatting verbs
%n: metadata.name, %s: metadata.namespace, %k: kind
`)
	return r
}

func NewCommand(parent string) *cobra.Command {
	return NewRunner(parent).Command
}

// Runner contains the run function
type Runner struct {
	Get             get.Command
	Command         *cobra.Command
	FilenamePattern string
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
		return pkgURI[0], "master", nil
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
			return "", errors.Errorf("parent directory %s does not exist", parent)
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
		return "", errors.Errorf("destination directory %s already exists", v)
	}
	return v, nil
}

func (r *Runner) preRunE(c *cobra.Command, args []string) error {
	if args[0] == "-" {
		return nil
	}

	// Simple parsing if contains .git
	if strings.Contains(args[0], ".git") {
		var repo, dir, version string
		parts := strings.Split(args[0], ".git")
		repo = strings.Trim(parts[0], "/")
		if len(parts) == 1 {
			// do nothing
		} else if strings.Contains(parts[1], "@") {
			parts := strings.Split(parts[1], "@")
			version = strings.Trim(parts[1], "/")
			dir = parts[0]
		} else {
			dir = parts[1]
		}
		if version == "" {
			version = "master"
		}
		if dir == "" {
			dir = "/"
		}
		destination, err := getDest(args[1], repo, dir)
		if err != nil {
			return err
		}
		r.Get.Ref = version
		r.Get.Directory = path.Clean(dir)
		r.Get.Repo = repo
		r.Get.Destination = filepath.Clean(destination)
		return nil
	}

	uri, version, err := getURIAndVersion(args[0])
	if err != nil {
		return err
	}
	repo, remoteDir, err := getRepoAndPkg(uri)
	if err != nil {
		return err
	}
	destination, err := getDest(args[1], repo, remoteDir)
	if err != nil {
		return err
	}
	r.Get.Ref = version
	r.Get.Directory = path.Clean(remoteDir)
	r.Get.Repo = repo
	r.Get.Destination = filepath.Clean(destination)
	return nil
}

func (r *Runner) runE(c *cobra.Command, args []string) error {
	if args[0] == "-" {
		return getioreader.Get(args[1], r.FilenamePattern, c.InOrStdin())
	}

	fmt.Fprintf(c.OutOrStdout(),
		"fetching package %s from %s to %s\n",
		r.Get.Directory, r.Get.Repo, r.Get.Destination)
	if err := r.Get.Run(); err != nil {
		return err
	}
	return nil
}
