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
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
)

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (kptfilev1alpha2.KptFile, error) {
	const op errors.Op = "kptfileutil.ReadFile"
	kpgfile := kptfilev1alpha2.KptFile{ResourceMeta: kptfilev1alpha2.TypeMeta}

	f, err := os.Open(filepath.Join(dir, kptfilev1alpha2.KptFileName))
	if err != nil {
		return kptfilev1alpha2.KptFile{}, errors.E(op, errors.IO, types.UniquePath(dir), err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(&kpgfile); err != nil {
		return kptfilev1alpha2.KptFile{}, errors.E(op, errors.YAML, types.UniquePath(dir), err)
	}
	return kpgfile, nil
}

func WriteFile(dir string, k kptfilev1alpha2.KptFile) error {
	const op errors.Op = "kptfileutil.WriteFile"
	b, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, kptfilev1alpha2.KptFileName)); err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}

	// convert to rNode and back to string to make indentation consistent
	// with rest of the yaml serialization to avoid unwanted diffs
	rNode, err := yaml.Parse(string(b))
	if err != nil {
		return errors.E(op, errors.YAML, types.UniquePath(dir), err)
	}

	kptFileStr, err := rNode.String()
	if err != nil {
		return errors.E(op, errors.YAML, types.UniquePath(dir), err)
	}

	// fyi: perm is ignored if the file already exists
	err = ioutil.WriteFile(filepath.Join(dir, kptfilev1alpha2.KptFileName), []byte(kptFileStr), 0600)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}
	return nil
}

// ValidateInventory returns true and a nil error if the passed inventory
// is valid; otherwise, false and the reason the inventory is not valid
// is returned. A valid inventory must have a non-empty namespace, name,
// and id.
func ValidateInventory(inv *kptfilev1alpha2.Inventory) (bool, error) {
	const op errors.Op = "kptfileutil.ValidateInventory"
	if inv == nil {
		return false, errors.E(op, errors.MissingParam,
			fmt.Errorf("kptfile missing inventory section"))
	}
	// Validate the name, namespace, and inventory id
	if strings.TrimSpace(inv.Name) == "" {
		return false, errors.E(op, errors.MissingParam,
			fmt.Errorf("kptfile inventory empty name"))
	}
	if strings.TrimSpace(inv.Namespace) == "" {
		return false, errors.E(op, errors.MissingParam,
			fmt.Errorf("kptfile inventory empty namespace"))
	}
	if strings.TrimSpace(inv.InventoryID) == "" {
		return false, errors.E(op, errors.MissingParam,
			fmt.Errorf("kptfile inventory missing inventoryID"))
	}
	return true, nil
}

func Equal(kf1, kf2 kptfilev1alpha2.KptFile) (bool, error) {
	const op errors.Op = "kptfileutil.Equal"
	kf1Bytes, err := yaml.Marshal(kf1)
	if err != nil {
		return false, errors.E(op, errors.YAML, err)
	}

	kf2Bytes, err := yaml.Marshal(kf2)
	if err != nil {
		return false, errors.E(op, errors.YAML, err)
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
	const op errors.Op = "kptfileutil.UpdateKptfileWithoutOrigin"
	localKf, err := ReadFile(localPath)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, types.UniquePath(localPath), err)
	}

	updatedKf, err := ReadFile(updatedPath)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, types.UniquePath(updatedPath), err)
	}

	localKf, err = merge(localKf, updatedKf, kptfilev1alpha2.KptFile{})
	if err != nil {
		return err
	}

	if updateUpstream {
		localKf = updateUpstreamAndUpstreamLock(localKf, updatedKf)
	}

	err = WriteFile(localPath, localKf)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	return nil
}

// UpdateKptfile updates the Kptfile in the package specified by localPath with
// values from the packages specified in updatedPath using the package specified
// by originPath as the common ancestor.
// If updateUpstream is true, the values from the upstream and upstreamLock
// sections will also be copied into local.
func UpdateKptfile(localPath, updatedPath, originPath string, updateUpstream bool) error {
	const op errors.Op = "kptfileutil.UpdateKptfile"
	localKf, err := ReadFile(localPath)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, types.UniquePath(localPath), err)
	}

	updatedKf, err := ReadFile(updatedPath)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, types.UniquePath(localPath), err)
	}

	originKf, err := ReadFile(originPath)
	if err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, types.UniquePath(localPath), err)
	}

	localKf, err = merge(localKf, updatedKf, originKf)
	if err != nil {
		return err
	}

	if updateUpstream {
		localKf = updateUpstreamAndUpstreamLock(localKf, updatedKf)
	}

	err = WriteFile(localPath, localKf)
	if err != nil {
		return errors.E(op, types.UniquePath(localPath), err)
	}
	return nil
}

// UpdateUpstreamLockFromGit updates the upstreamLock of the package specified
// by path by using the values from spec. It will also populate the commit
// field in upstreamLock using the latest commit of the git repo given
// by spec.
func UpdateUpstreamLockFromGit(path string, spec *git.RepoSpec) error {
	const op errors.Op = "kptfileutil.UpdateUpstreamLockFromGit"
	// read KptFile cloned with the package if it exists
	kpgfile, err := ReadFile(path)
	if err != nil {
		return errors.E(op, types.UniquePath(path), err)
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.UpstreamLock = &kptfilev1alpha2.UpstreamLock{
		Type: kptfilev1alpha2.GitOrigin,
		GitLock: &kptfilev1alpha2.GitLock{
			Repo:      spec.OrgRepo,
			Directory: spec.Path,
			Ref:       spec.Ref,
			Commit:    spec.Commit,
		},
	}
	err = WriteFile(path, kpgfile)
	if err != nil {
		return errors.E(op, types.UniquePath(path), err)
	}
	return nil
}

func merge(localKf, updatedKf, originalKf kptfilev1alpha2.KptFile) (kptfilev1alpha2.KptFile, error) {
	localBytes, err := yaml.Marshal(localKf)
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
	}

	updatedBytes, err := yaml.Marshal(updatedKf)
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
	}

	originalBytes, err := yaml.Marshal(originalKf)
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
	}

	mergedBytes, err := merge3.MergeStrings(string(localBytes), string(originalBytes), string(updatedBytes), false)
	if err != nil {
		return kptfilev1alpha2.KptFile{}, err
	}

	var mergedKf kptfilev1alpha2.KptFile
	err = yaml.Unmarshal([]byte(mergedBytes), &mergedKf)
	if err != nil {
		return mergedKf, err
	}

	// The merge algorithm currently lets values from upstream take precedence,
	// and we don't want that for name and namespace. So updating those values
	// in the merge Kptfile.
	mergedKf.ObjectMeta.Name = localKf.Name
	mergedKf.ObjectMeta.Namespace = localKf.Namespace

	// We don't want the values from upstream here, so we set the values back
	// to what was already in local.
	mergedKf.Upstream = nil
	mergedKf.UpstreamLock = nil

	if localKf.Upstream != nil {
		mergedKf.Upstream = &kptfilev1alpha2.Upstream{
			Type: localKf.Upstream.Type,
			Git: &kptfilev1alpha2.Git{
				Directory: localKf.Upstream.Git.Directory,
				Repo:      localKf.Upstream.Git.Repo,
				Ref:       localKf.Upstream.Git.Ref,
			},
			UpdateStrategy: localKf.Upstream.UpdateStrategy,
		}
	}

	if localKf.UpstreamLock != nil {
		mergedKf.UpstreamLock = &kptfilev1alpha2.UpstreamLock{
			Type: localKf.UpstreamLock.Type,
			GitLock: &kptfilev1alpha2.GitLock{
				Commit:    localKf.UpstreamLock.GitLock.Commit,
				Directory: localKf.UpstreamLock.GitLock.Directory,
				Repo:      localKf.UpstreamLock.GitLock.Repo,
				Ref:       localKf.UpstreamLock.GitLock.Ref,
			},
		}
	}
	return mergedKf, nil
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
