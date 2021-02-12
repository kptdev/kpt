// Copyright 2020 Google LLC
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

package diff

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1alpha2 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
	"github.com/GoogleContainerTools/kpt/pkg/kptfile/kptfileutil"
	"sigs.k8s.io/kustomize/kyaml/sets"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func PkgDiff(pkg1, pkg2 string) (sets.String, error) {
	pkg1Files, err := pkgSet(pkg1)
	if err != nil {
		return sets.String{}, err
	}

	pkg2Files, err := pkgSet(pkg2)
	if err != nil {
		return sets.String{}, err
	}

	diff := pkg1Files.SymmetricDifference(pkg2Files)

	for _, f := range pkg1Files.Intersection(pkg2Files).List() {
		fi, err := os.Stat(filepath.Join(pkg1, f))
		if err != nil {
			return diff, err
		}

		if fi.IsDir() {
			continue
		}

		fileName := filepath.Base(f)
		if fileName == kptfilev1alpha2.KptFileName {
			equal, err := kptfilesEqual(pkg1, pkg2, f)
			if err != nil {
				return diff, err
			}
			if !equal {
				diff.Insert(f)
			}
		} else {
			b1, err := ioutil.ReadFile(filepath.Join(pkg1, f))
			if err != nil {
				return diff, err
			}
			b2, err := ioutil.ReadFile(filepath.Join(pkg2, f))
			if err != nil {
				return diff, err
			}
			if !bytes.Equal(b1, b2) {
				diff.Insert(f)
			}
		}
	}
	return diff, nil
}

func kptfilesEqual(pkg1, pkg2, filePath string) (bool, error) {
	pkg1Kf, err := kptfileutil.ReadFile(filepath.Join(pkg1, filepath.Dir(filePath)))
	if err != nil {
		return false, err
	}
	pkg2Kf, err := kptfileutil.ReadFile(filepath.Join(pkg2, filepath.Dir(filePath)))
	if err != nil {
		return false, err
	}

	pkg1Kf.UpstreamLock = &kptfilev1alpha2.UpstreamLock{}
	pkg2Kf.UpstreamLock = &kptfilev1alpha2.UpstreamLock{}

	pkg1Bytes, err := yaml.Marshal(pkg1Kf)
	if err != nil {
		return false, err
	}
	pkg2Bytes, err := yaml.Marshal(pkg2Kf)
	if err != nil {
		return false, err
	}
	return bytes.Equal(pkg1Bytes, pkg2Bytes), nil
}

func pkgSet(pkgPath string) (sets.String, error) {
	pkgFiles := sets.String{}
	if err := pkgutil.WalkPackage(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(pkgPath, path)
		if err != nil {
			return err
		}
		pkgFiles.Insert(relPath)
		return nil
	}); err != nil {
		return sets.String{}, err
	}
	return pkgFiles, nil
}
