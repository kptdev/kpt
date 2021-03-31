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
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (kptfilev1alpha2.KptFile, error) {
	kpgfile := kptfilev1alpha2.KptFile{ResourceMeta: kptfilev1alpha2.TypeMeta}

	f, err := os.Open(filepath.Join(dir, kptfilev1alpha2.KptFileName))
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
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

// UpdateKptfileWithoutOrigin updates the Kptfile in the package specified by
// localPath with values from the package specified by updatedPath using a 3-way
// merge strategy, but where origin does not have any values.
// If updateUpstream is true, the values from the upstream and upstreamLock
// sections will also be copied into local.
func UpdateKptfileWithoutOrigin(localPath, updatedPath string, updateUpstream bool) error {
	localKf, err := ReadFile(localPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updatedKf, err := ReadFile(updatedPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// TODO: Merge other parts of the Kptfile

	if updateUpstream {
		localKf = updateUpstreamAndUpstreamLock(localKf, updatedKf)
	}

	return WriteFile(localPath, localKf)
}

// UpdateKptfile updates the Kptfile in the package specified by localPath with
// values from the packages specified in updatedPath using the package specified
// by originPath as the common ancestor.
// If updateUpstream is true, the values from the upstream and upstreamLock
// sections will also be copied into local.
func UpdateKptfile(localPath, updatedPath, originPath string, updateUpstream bool) error {
	localKf, err := ReadFile(localPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	updatedKf, err := ReadFile(updatedPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// TODO: Merge other parts of the Kptfile

	if updateUpstream {
		localKf = updateUpstreamAndUpstreamLock(localKf, updatedKf)
	}

	return WriteFile(localPath, localKf)
}

// UpdateUpstreamLockFromGit updates the upstreamLock of the package specified
// by path by using the values from spec. It will also populate the commit
// field in upstreamLock using the latest commit of the git repo given
// by spec.
func UpdateUpstreamLockFromGit(path string, spec *git.RepoSpec) error {
	// read KptFile cloned with the package if it exists
	kpgfile, err := ReadFile(path)
	if err != nil {
		return err
	}

	// find the git commit sha that we cloned the package at so we can write it to the KptFile
	commit, err := git.LookupCommit(spec.AbsPath())
	if err != nil {
		return err
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.UpstreamLock = &kptfilev1alpha2.UpstreamLock{
		Type: kptfilev1alpha2.GitOrigin,
		GitLock: &kptfilev1alpha2.GitLock{
			Repo:      spec.OrgRepo,
			Directory: spec.Path,
			Ref:       spec.Ref,
			Commit:    commit,
		},
	}
	return WriteFile(path, kpgfile)
}

func updateUpstreamAndUpstreamLock(localKf, updatedKf kptfilev1alpha2.KptFile) kptfilev1alpha2.KptFile {
	if updatedKf.Upstream != nil {
		localKf.Upstream = &kptfilev1alpha2.Upstream{
			Type: updatedKf.Upstream.Type,
			Git: &kptfilev1alpha2.Git{
				Directory: updatedKf.Upstream.Git.Directory,
				Repo:      updatedKf.Upstream.Git.Repo,
				Ref:       updatedKf.Upstream.Git.Ref,
			},
			UpdateStrategy: updatedKf.Upstream.UpdateStrategy,
		}
	}

	if updatedKf.UpstreamLock != nil {
		localKf.UpstreamLock = &kptfilev1alpha2.UpstreamLock{
			Type: updatedKf.UpstreamLock.Type,
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    updatedKf.UpstreamLock.GitLock.Commit,
				Directory: updatedKf.UpstreamLock.GitLock.Directory,
				Repo:      updatedKf.UpstreamLock.GitLock.Repo,
				Ref:       updatedKf.UpstreamLock.GitLock.Ref,
			},
		}
	}
	return localKf
}
