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

// Package sync syncs dependencies specified in the Kptfile to local directories.
package sync

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/kptfile"
	"github.com/GoogleContainerTools/kpt/internal/util/get"
	"github.com/GoogleContainerTools/kpt/internal/util/update"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Command syncs the dependencies delared in the Kptfile -- getting, updating, or
// deleting the dependencies as needed.
type Command struct {
	// Dir is the path to a directory containing the Kptfile to sync
	Dir string

	Verbose bool
	DryRun  bool
	StdOut  io.Writer
	StdErr  io.Writer
}

// Run syncs all dependencies declared in the Kptfile, fetching them
// if they are missing, updating them if their versions have changed,
// and deleting them if they should not exist.
func (c Command) Run() error {
	b, err := ioutil.ReadFile(filepath.Join(c.Dir, kptfile.KptFileName))
	if err != nil {
		return errors.WrapPrefixf(err, "failed to read Kptfile under %s", c.Dir)
	}
	k := &kptfile.KptFile{}

	if err := yaml.Unmarshal(b, k); err != nil {
		return errors.WrapPrefixf(err, "failed to unmarshal Kptfile under %s", c.Dir)
	}

	// validate dependencies are well formed
	for i := range k.Dependencies {
		if k.Dependencies[i].Name == "" {
			return errors.Errorf("One or more dependencies missing 'name'")
		}
		if k.Dependencies[i].Path == "" {
			return errors.Errorf("One or more dependencies missing 'path'")
		}
		if !k.Dependencies[i].EnsureNotExists {
			if k.Dependencies[i].Git.Directory == "" {
				return errors.Errorf("One or more dependencies missing 'git.directory'")
			}
			if k.Dependencies[i].Git.Ref == "" {
				return errors.Errorf("One or more dependencies missing 'git.ref'")
			}
			if k.Dependencies[i].Git.Repo == "" {
				return errors.Errorf("One or more dependencies missing 'git.repo'")
			}
		} else {
			if k.Dependencies[i].Git.Directory != "" ||
				k.Dependencies[i].Git.Ref != "" ||
				k.Dependencies[i].Git.Repo == "" {
				return errors.Errorf(
					"One or more dependencies specify mutually exclusive fields " +
						"'ensureNotExists' and 'git'")
			}
		}
	}

	for i := range k.Dependencies {
		if err := c.sync(k.Dependencies[i]); err != nil {
			return err
		}
	}
	return nil
}

func (c Command) sync(dependency kptfile.Dependency) error {
	path := filepath.Join(c.Dir, dependency.Path)
	f, err := os.Stat(path)

	// dep missing
	if os.IsNotExist(err) {
		if dependency.EnsureNotExists {
			// dep already deleted -- no action required
			return nil
		}
		// fetch the dep
		return c.get(dependency)
	}
	if err != nil {
		return errors.Wrap(err)
	}

	// verify the dep is well formed
	if !f.IsDir() {
		// place where dep should be fetched exists and is not a directory
		return errors.Errorf("cannot sync to %s, non-direcotry file exists", path)
	}
	_, err = os.Stat(filepath.Join(path, kptfile.KptFileName))
	if os.IsNotExist(err) {
		// dep doesn't have a Kptfile -- something is wrong
		return errors.Errorf("expected Kptfile under dependency %s", path)
	}
	if err != nil {
		return errors.Wrap(err)
	}

	// delete the dep -- it exists
	if dependency.EnsureNotExists {
		return c.delete(dependency)
	}

	// read the Kptfile
	b, err := ioutil.ReadFile(filepath.Join(path, kptfile.KptFileName))
	if err != nil {
		return errors.Wrap(err)
	}
	k := &kptfile.KptFile{}
	if err = yaml.Unmarshal(b, k); err != nil {
		return errors.Wrap(err)
	}

	return c.update(dependency, k)
}

// get fetches the dependency
func (c Command) get(dependency kptfile.Dependency) error {
	path := filepath.Join(c.Dir, dependency.Path)
	fmt.Fprintf(c.StdOut, "fetching %s (%s)\n", dependency.Name, path)
	if c.DryRun {
		return nil
	}

	return get.Command{
		Git:         dependency.Git,
		Destination: path,
		Name:        dependency.Name,
	}.Run()
}

// update updates the version of the fetched dependency to match
func (c Command) update(dependency kptfile.Dependency, k *kptfile.KptFile) error {
	path := filepath.Join(c.Dir, dependency.Path)
	fmt.Fprintf(c.StdOut, "updating %s (%s) from %s to %s\n",
		dependency.Name, path, k.Upstream.Git.Ref, dependency.Git.Ref)
	if c.DryRun {
		return nil
	}

	if dependency.Git.Ref == k.Upstream.Git.Ref &&
		dependency.Git.Repo == k.Upstream.Git.Repo {
		return nil
	}
	return update.Command{
		Path:     path,
		Ref:      dependency.Git.Ref,
		Repo:     dependency.Git.Repo,
		Strategy: update.StrategyType(dependency.Strategy),
		Verbose:  c.Verbose,
	}.Run()
}

// delete removes the dependency if it exists
func (c Command) delete(dependency kptfile.Dependency) error {
	path := filepath.Join(c.Dir, dependency.Path)
	fmt.Fprintf(c.StdOut, "deleting %s (%s)\n", dependency.Name, path)
	if c.DryRun {
		return nil
	}

	return os.RemoveAll(path)
}
