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
	"strings"

	"github.com/GoogleContainerTools/kpt/pkg/kptfile"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (kptfile.KptFile, error) {
	kpgfile := kptfile.KptFile{ResourceMeta: kptfile.TypeMeta}

	f, err := os.Open(filepath.Join(dir, kptfile.KptFileName))

	// if we are in a package subdirectory, find the parent dir with the Kptfile.
	// this is necessary to parse the duck-commands for sub-directories of a package
	for os.IsNotExist(err) && filepath.Base(dir) == kptfile.KptFileName {
		dir = filepath.Dir(dir)
		f, err = os.Open(filepath.Join(dir, kptfile.KptFileName))
	}
	if err != nil {
		return kptfile.KptFile{}, errors.Errorf("unable to read %s: %v", kptfile.KptFileName, err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(&kpgfile); err != nil {
		return kptfile.KptFile{}, errors.Errorf("unable to parse %s: %v", kptfile.KptFileName, err)
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

	// convert to rNode and back to string to make indentation consistent
	// with rest of the yaml serialization to avoid unwanted diffs
	rNode, err := yaml.Parse(string(b))
	if err != nil {
		return err
	}

	kptFileStr, err := rNode.String()
	if err != nil {
		return err
	}

	// fyi: perm is ignored if the file already exists
	return ioutil.WriteFile(filepath.Join(dir, kptfile.KptFileName), []byte(kptFileStr), 0600)
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
			return kptfile.KptFile{}, errors.Errorf("%s Kptfile missing upstream.git.repo", pkgPath)
		}
		if git.Commit == "" {
			return kptfile.KptFile{}, errors.Errorf("%s Kptfile missing upstream.git.commit", pkgPath)
		}
		if git.Ref == "" {
			return kptfile.KptFile{}, errors.Errorf("%s Kptfile missing upstream.git.ref", pkgPath)
		}
		if git.Directory == "" {
			return kptfile.KptFile{}, errors.Errorf("%s Kptfile missing upstream.git.directory", pkgPath)
		}
	}
	if kf.Upstream.Type == kptfile.StdinOrigin {
		stdin := kf.Upstream.Stdin
		if stdin.FilenamePattern == "" {
			return kptfile.KptFile{}, errors.Errorf(
				"%s Kptfile missing upstream.stdin.filenamePattern", pkgPath)
		}
		if stdin.Original == "" {
			return kptfile.KptFile{}, errors.Errorf(
				"%s Kptfile missing upstream.stdin.original", pkgPath)
		}
	}
	return kf, nil
}

// ValidateInventory returns true and a nil error if the passed inventory
// is valid; otherwise, false and the reason the inventory is not valid
// is returned. A valid inventory must have a non-empty namespace, name,
// and id.
func ValidateInventory(inv *kptfile.Inventory) (bool, error) {
	if inv == nil {
		return false, fmt.Errorf("kptfile missing inventory section")
	}
	// Validate the name, namespace, and inventory id
	if strings.TrimSpace(inv.Name) == "" {
		return false, fmt.Errorf("kptfile inventory empty name")
	}
	if strings.TrimSpace(inv.Namespace) == "" {
		return false, fmt.Errorf("kptfile inventory empty namespace")
	}
	if strings.TrimSpace(inv.InventoryID) == "" {
		return false, fmt.Errorf("kptfile inventory missing inventoryID")
	}
	return true, nil
}
