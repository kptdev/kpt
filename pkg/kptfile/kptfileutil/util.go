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
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge3"
)

func WriteFile(dir string, k *kptfilev1.KptFile) error {
	const op errors.Op = "kptfileutil.WriteFile"
	b, err := yaml.MarshalWithOptions(k, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, kptfilev1.KptFileName)); err != nil && !goerrors.Is(err, os.ErrNotExist) {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}

	// fyi: perm is ignored if the file already exists
	err = ioutil.WriteFile(filepath.Join(dir, kptfilev1.KptFileName), b, 0600)
	if err != nil {
		return errors.E(op, errors.IO, types.UniquePath(dir), err)
	}
	return nil
}

// ValidateInventory returns true and a nil error if the passed inventory
// is valid; otherwiste, false and the reason the inventory is not valid
// is returned. A valid inventory must have a non-empty namespace, name,
// and id.
func ValidateInventory(inv *kptfilev1.Inventory) (bool, error) {
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

func Equal(kf1, kf2 *kptfilev1.KptFile) (bool, error) {
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
func DefaultKptfile(name string) *kptfilev1.KptFile {
	return &kptfilev1.KptFile{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: kptfilev1.TypeMeta.APIVersion,
				Kind:       kptfilev1.TypeMeta.Kind,
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
	localKf, err := pkg.ReadKptfile(localPath)
	if err != nil {
		if !goerrors.Is(err, os.ErrNotExist) {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		localKf = &kptfilev1.KptFile{}
	}

	updatedKf, err := pkg.ReadKptfile(updatedPath)
	if err != nil {
		if !goerrors.Is(err, os.ErrNotExist) {
			return errors.E(op, types.UniquePath(updatedPath), err)
		}
		updatedKf = &kptfilev1.KptFile{}
	}

	err = merge(localKf, updatedKf, &kptfilev1.KptFile{})
	if err != nil {
		return err
	}

	if updateUpstream {
		updateUpstreamAndUpstreamLock(localKf, updatedKf)
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
	localKf, err := pkg.ReadKptfile(localPath)
	if err != nil {
		if !goerrors.Is(err, os.ErrNotExist) {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		localKf = &kptfilev1.KptFile{}
	}

	updatedKf, err := pkg.ReadKptfile(updatedPath)
	if err != nil {
		if !goerrors.Is(err, os.ErrNotExist) {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		updatedKf = &kptfilev1.KptFile{}
	}

	originKf, err := pkg.ReadKptfile(originPath)
	if err != nil {
		if !goerrors.Is(err, os.ErrNotExist) {
			return errors.E(op, types.UniquePath(localPath), err)
		}
		originKf = &kptfilev1.KptFile{}
	}

	err = merge(localKf, updatedKf, originKf)
	if err != nil {
		return err
	}

	if updateUpstream {
		updateUpstreamAndUpstreamLock(localKf, updatedKf)
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
	kpgfile, err := pkg.ReadKptfile(path)
	if err != nil {
		return errors.E(op, types.UniquePath(path), err)
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.UpstreamLock = &kptfilev1.UpstreamLock{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.GitLock{
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

func merge(localKf, updatedKf, originalKf *kptfilev1.KptFile) error {
	localBytes, err := yaml.Marshal(localKf)
	if err != nil {
		return err
	}

	updatedBytes, err := yaml.Marshal(updatedKf)
	if err != nil {
		return err
	}

	originalBytes, err := yaml.Marshal(originalKf)
	if err != nil {
		return err
	}

	mergedBytes, err := merge3.MergeStrings(string(localBytes), string(originalBytes), string(updatedBytes), false)
	if err != nil {
		return err
	}

	var mergedKf kptfilev1.KptFile
	err = yaml.Unmarshal([]byte(mergedBytes), &mergedKf)
	if err != nil {
		return err
	}

	// Copy the merged content into the local Kptfile struct. We don't copy
	// name, namespace, Upstream or UpstreamLock, since we don't want those
	// merged.
	localKf.Annotations = mergedKf.Annotations
	localKf.Labels = mergedKf.Labels
	localKf.Info = mergedKf.Info
	localKf.Pipeline = mergedKf.Pipeline
	localKf.Inventory = mergedKf.Inventory
	return nil
}

func updateUpstreamAndUpstreamLock(localKf, updatedKf *kptfilev1.KptFile) {
	if updatedKf.Upstream != nil {
		localKf.Upstream = updatedKf.Upstream
	}

	if updatedKf.UpstreamLock != nil {
		localKf.UpstreamLock = updatedKf.UpstreamLock
	}
}
