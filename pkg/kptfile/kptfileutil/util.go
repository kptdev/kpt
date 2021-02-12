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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (kptfilev1alpha2.KptFile, error) {
	kpgfile := kptfilev1alpha2.KptFile{ResourceMeta: kptfilev1alpha2.TypeMeta}

	f, err := os.Open(filepath.Join(dir, kptfilev1alpha2.KptFileName))

	// if we are in a package subdirectory, find the parent dir with the Kptfile.
	// this is necessary to parse the duck-commands for sub-directories of a package
	for os.IsNotExist(err) && filepath.Base(dir) == kptfilev1alpha2.KptFileName {
		dir = filepath.Dir(dir)
		f, err = os.Open(filepath.Join(dir, kptfilev1alpha2.KptFileName))
	}
	if err != nil {
		return kptfilev1alpha2.KptFile{}, errors.Errorf("unable to read %s: %v", kptfilev1alpha2.KptFileName, err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(&kpgfile); err != nil {
		return kptfilev1alpha2.KptFile{}, errors.Errorf("unable to parse %s: %v", kptfilev1alpha2.KptFileName, err)
	}
	return kpgfile, nil
}

func WriteFile(dir string, k kptfilev1alpha2.KptFile) error {
	b, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, kptfilev1alpha2.KptFileName)); err != nil && !os.IsNotExist(err) {
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
	return ioutil.WriteFile(filepath.Join(dir, kptfilev1alpha2.KptFileName), []byte(kptFileStr), 0600)
}

// ReadFileStrict reads a Kptfile for a package and validates that it contains required
// Upstream fields.
func ReadFileStrict(pkgPath string) (kptfilev1alpha2.KptFile, error) {
	kf, err := ReadFile(pkgPath)
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
	}

	if kf.UpstreamLock != nil && kf.UpstreamLock.Type == kptfilev1alpha2.GitOrigin {
		git := kf.UpstreamLock.GitLock
		if git.Repo == "" {
			return kptfilev1alpha2.KptFile{}, errors.Errorf("%s Kptfile missing upstreamLock.gitLock.repo", pkgPath)
		}
		if git.Commit == "" {
			return kptfilev1alpha2.KptFile{}, errors.Errorf("%s Kptfile missing upstreamLock.gitLock.commit", pkgPath)
		}
		if git.Ref == "" {
			return kptfilev1alpha2.KptFile{}, errors.Errorf("%s Kptfile missing upstreamLock.gitLock.ref", pkgPath)
		}
		if git.Directory == "" {
			return kptfilev1alpha2.KptFile{}, errors.Errorf("%s Kptfile missing upstreamLock.gitLock.directory", pkgPath)
		}
	}
	return kf, nil
}

// ValidateInventory returns true and a nil error if the passed inventory
// is valid; otherwise, false and the reason the inventory is not valid
// is returned. A valid inventory must have a non-empty namespace, name,
// and id.
func ValidateInventory(inv *kptfilev1alpha2.Inventory) (bool, error) {
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

func Equal(kf1, kf2 kptfilev1alpha2.KptFile) (bool, error) {
	kf1Bytes, err := yaml.Marshal(kf1)
	if err != nil {
		return false, err
	}

	kf2Bytes, err := yaml.Marshal(kf2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(kf1Bytes, kf2Bytes), nil
}

// DefaultKptfile returns a new minimal Kptfile.
func DefaultKptfile(name string) kptfilev1alpha2.KptFile {
	return kptfilev1alpha2.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1alpha2.TypeMeta.APIVersion,
				Kind:       kptfilev1alpha2.TypeMeta.Kind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: name,
				},
			},
		},
	}
}

// HasKptfile checks if there exists a Kptfile on the provided path.
func HasKptfile(path string) (bool, error) {
	_, err := os.Stat(filepath.Join(path, kptfilev1alpha2.KptFileName))

	// If we got an error that wasn't IsNotExist, something went wrong and
	// we don't really know if the file exists or not.
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	// If the error is IsNotExist, we know the file doesn't exist.
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

// MergeSubpackages takes the subpackage information from local, updated
// and original and does a 3-way merge. The result is returned as a new slice.
// The passed in data structures are not changed.
func MergeSubpackages(local, updated, original []kptfilev1alpha2.Subpackage) ([]kptfilev1alpha2.Subpackage, error) {
	// find is a helper function that returns a subpackage with the provided
	// key from the slice.
	find := func(key string, slice []kptfilev1alpha2.Subpackage) (kptfilev1alpha2.Subpackage, bool) {
		for i := range slice {
			sp := slice[i]
			if sp.LocalDir == key {
				return sp, true
			}
		}
		return kptfilev1alpha2.Subpackage{}, false
	}

	// Create a new slice to contain the merged result.
	var merged []kptfilev1alpha2.Subpackage

	// Create a slice that contains all keys available from both updated
	// and local. We add keys from updated first, so subpackages added
	// locally will be at the end of the slice after merge.
	var dirKeys []string
	for _, sp := range updated {
		dirKeys = append(dirKeys, sp.LocalDir)
	}
	for _, sp := range local {
		dirKeys = append(dirKeys, sp.LocalDir)
	}

	// The slice of keys might contain duplicates, so keep track of which
	// keys we have seen.
	seen := make(map[string]bool)
	for _, key := range dirKeys {
		// Skip subpackages that we have already merged.
		if seen[key] {
			continue
		}
		seen[key] = true

		// Look up the package with the given name from all three sources.
		l, lok := find(key, local)
		u, uok := find(key, updated)
		o, ook := find(key, original)

		// If we find a remote subpackage defined in both local and updated, but
		// not in the original, it must have been added both in local and updated.
		// This is an error and the user must resolve this.
		if !ook && uok && lok {
			return merged, fmt.Errorf("remote subpackage with localDir %s added in both local and upstream", key)
		}

		// If not in either upstream or local, we don't need to add it.
		if !lok && !uok {
			continue
		}

		// If deleted from upstream, we only remove it if local is unchanged.
		if ook && !uok {
			if !reflect.DeepEqual(o, l) {
				merged = append(merged, l)
			}
			continue
		}

		// If deleted from local, we don't want to re-add it from upstream.
		if ook && !lok {
			continue
		}

		// If key not found in local, we use the version from updated.
		if !lok {
			merged = append(merged, u)
			continue
		}
		// If key not found in updated, we use the version from local.
		if !uok {
			merged = append(merged, l)
			continue
		}

		// If we changes to local compared with original, we keep the local
		// version. Otherwise, we take hte version from updated.
		if reflect.DeepEqual(o, l) {
			merged = append(merged, u)
		} else {
			merged = append(merged, l)
		}
	}
	return merged, nil
}
