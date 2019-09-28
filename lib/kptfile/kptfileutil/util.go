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

package kptfileutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/yaml"
)

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (kptfile.KptFile, error) {
	kpgfile := kptfile.KptFile{ResourceMeta: kptfile.TypeMeta}

	f, err := os.Open(filepath.Join(dir, kptfile.KptFileName))

	// if we are in a package subdirectory, find the parent dir with the Kptfile.
	// this is necessary to parse the duck-commands for sub-directories of a package
	for os.IsNotExist(err) && filepath.Dir(dir) != dir {
		dir = filepath.Dir(dir)
		f, err = os.Open(filepath.Join(dir, kptfile.KptFileName))
	}
	if err != nil {
		return kptfile.KptFile{}, fmt.Errorf("unable to read %s: %v", kptfile.KptFileName, err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(&kpgfile); err != nil {
		return kptfile.KptFile{}, fmt.Errorf("unable to parse %s: %v", kptfile.KptFileName, err)
	}
	return kpgfile, nil
}

func WriteFile(dir string, k kptfile.KptFile) error {
	b, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, kptfile.KptFileName)); err != nil && !os.IsNotExist(err) {
		return err
	}

	// fyi: perm is ignored if the file already exists
	return ioutil.WriteFile(filepath.Join(dir, kptfile.KptFileName), b, 0600)
}

// ReadFileStrict reads a Kptfile for a package and validates that it contains required
// Upstream fields.
func ReadFileStrict(pkgPath string) (kptfile.KptFile, error) {
	kf, err := ReadFile(pkgPath)
	if err != nil {
		return kptfile.KptFile{}, err
	}

	if kf.Upstream.Type == kptfile.GitOrigin {
		git := kf.Upstream.Git
		if git.Repo == "" {
			return kptfile.KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.repo", pkgPath)
		}
		if git.Commit == "" {
			return kptfile.KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.commit", pkgPath)
		}
		if git.Ref == "" {
			return kptfile.KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.ref", pkgPath)
		}
		if git.Directory == "" {
			return kptfile.KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.directory", pkgPath)
		}
	}
	if kf.Upstream.Type == kptfile.StdinOrigin {
		stdin := kf.Upstream.Stdin
		if stdin.FilenamePattern == "" {
			return kptfile.KptFile{}, fmt.Errorf(
				"%s Kptfile missing upstream.stdin.filenamePattern", pkgPath)
		}
		if stdin.Original == "" {
			return kptfile.KptFile{}, fmt.Errorf(
				"%s Kptfile missing upstream.stdin.original", pkgPath)
		}
	}
	return kf, nil
}
