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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/types"
	"github.com/GoogleContainerTools/kpt/internal/util/git"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/sets"
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

func Write(dst io.Writer, k *kptfilev1.KptFile) error {
	const op errors.Op = "kptfileutil.WriteFile"
	b, err := yaml.MarshalWithOptions(k, &yaml.EncoderOptions{SeqIndent: yaml.WideSequenceStyle})
	if err != nil {
		return err
	}

	if _, err := dst.Write(b); err != nil {
		return errors.E(op, errors.IO, err)
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

func UpdateUpstreamLock(path string, upstreamLock *kptfilev1.UpstreamLock) error {
	const op errors.Op = "kptfileutil.UpdateUpstreamLock"
	// read KptFile cloned with the package if it exists
	kpgfile, err := pkg.ReadKptfile(path)
	if err != nil {
		return errors.E(op, types.UniquePath(path), err)
	}

	// populate the cloneFrom values so we know where the package came from
	kpgfile.UpstreamLock = upstreamLock

	err = WriteFile(path, kpgfile)
	if err != nil {
		return errors.E(op, types.UniquePath(path), err)
	}
	return nil
}

// UpdateUpstreamLockFromGit updates the upstreamLock of the package specified
// by path by using the values from spec. It will also populate the commit
// field in upstreamLock using the latest commit of the git repo given
// by spec.
func UpdateUpstreamLockFromGit(path string, spec *git.RepoSpec) error {
	return UpdateUpstreamLock(path, &kptfilev1.UpstreamLock{
		Type: kptfilev1.GitOrigin,
		Git: &kptfilev1.GitLock{
			Repo:      spec.OrgRepo,
			Directory: spec.Path,
			Ref:       spec.Ref,
			Commit:    spec.Commit,
		},
	})
}

// merge merges the Kptfiles from various sources and updates localKf with output
// please refer to https://github.com/GoogleContainerTools/kpt/blob/main/docs/design-docs/03-pipeline-merge.md
// for related design
func merge(localKf, updatedKf, originalKf *kptfilev1.KptFile) error {
	shouldAddSyntheticMergeName := shouldAddFnKey(localKf, updatedKf, originalKf)
	if shouldAddSyntheticMergeName {
		addNameForMerge(localKf, updatedKf, originalKf)
	}

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

	mergedBytes, err := merge3.MergeStrings(string(localBytes), string(originalBytes), string(updatedBytes), true)
	if err != nil {
		return err
	}

	var mergedKf kptfilev1.KptFile
	err = yaml.Unmarshal([]byte(mergedBytes), &mergedKf)
	if err != nil {
		return err
	}

	if shouldAddSyntheticMergeName {
		removeFnKey(localKf, updatedKf, originalKf, &mergedKf)
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

// shouldAddFnKey returns true iff all the functions from all sources
// doesn't have name field set and there are no duplicate function declarations,
// it means the user is unaware of name field, and we use image name or exec field
// value as mergeKey instead of name in such cases
func shouldAddFnKey(kfs ...*kptfilev1.KptFile) bool {
	for _, kf := range kfs {
		if kf == nil || kf.Pipeline == nil {
			continue
		}
		if !shouldAddFnKeyUtil(kf.Pipeline.Mutators) || !shouldAddFnKeyUtil(kf.Pipeline.Validators) {
			return false
		}
	}
	return true
}

// shouldAddFnKeyUtil returns true iff all the functions from input list
// doesn't have name field set and there are no duplicate function declarations,
// it means the user is unaware of name field, and we use image name or exec field
// value as mergeKey instead of name in such cases
func shouldAddFnKeyUtil(fns []kptfilev1.Function) bool {
	keySet := sets.String{}
	for _, fn := range fns {
		if fn.Name != "" {
			return false
		}
		var key string
		if fn.Exec != "" {
			key = fn.Exec
		} else {
			key = strings.Split(fn.Image, ":")[0]
		}
		if keySet.Has(key) {
			return false
		}
		keySet.Insert(key)
	}
	return true
}

// addNameForMerge adds name field for all the functions if empty
// name is primarily used as merge-key
func addNameForMerge(kfs ...*kptfilev1.KptFile) {
	for _, kf := range kfs {
		if kf == nil || kf.Pipeline == nil {
			continue
		}
		for i, mutator := range kf.Pipeline.Mutators {
			kf.Pipeline.Mutators[i] = addName(mutator)
		}
		for i, validator := range kf.Pipeline.Validators {
			kf.Pipeline.Validators[i] = addName(validator)
		}
	}
}

// addName adds name field to the input function if empty
// name is nothing but image name in this case as we use it as fall back mergeKey
func addName(fn kptfilev1.Function) kptfilev1.Function {
	if fn.Name != "" {
		return fn
	}
	var key string
	if fn.Exec != "" {
		key = fn.Exec
	} else {
		parts := strings.Split(fn.Image, ":")
		if len(parts) > 0 {
			key = parts[0]
		}
	}
	fn.Name = fmt.Sprintf("_kpt-merge_%s", key)
	return fn
}

// removeFnKey remove the synthesized function name field before writing
func removeFnKey(kfs ...*kptfilev1.KptFile) {
	for _, kf := range kfs {
		if kf == nil || kf.Pipeline == nil {
			continue
		}
		for i := range kf.Pipeline.Mutators {
			if strings.HasPrefix(kf.Pipeline.Mutators[i].Name, "_kpt-merge_") {
				kf.Pipeline.Mutators[i].Name = ""
			}
		}
		for i := range kf.Pipeline.Validators {
			if strings.HasPrefix(kf.Pipeline.Validators[i].Name, "_kpt-merge_") {
				kf.Pipeline.Validators[i].Name = ""
			}
		}
	}
}

func updateUpstreamAndUpstreamLock(localKf, updatedKf *kptfilev1.KptFile) {
	if updatedKf.Upstream != nil {
		localKf.Upstream = updatedKf.Upstream
	}

	if updatedKf.UpstreamLock != nil {
		localKf.UpstreamLock = updatedKf.UpstreamLock
	}
}
