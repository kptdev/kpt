// Copyright 2020 The kpt Authors
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
	"os"
	"path/filepath"

	"github.com/GoogleContainerTools/kpt/internal/pkg"
	"github.com/GoogleContainerTools/kpt/internal/util/attribution"
	"github.com/GoogleContainerTools/kpt/internal/util/pkgutil"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
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
		if fileName == kptfilev1.KptFileName {
			equal, err := kptfilesEqual(pkg1, pkg2, f)
			if err != nil {
				return diff, err
			}
			if !equal {
				diff.Insert(f)
			}
		} else {
			b1, err := os.ReadFile(filepath.Join(pkg1, f))
			if err != nil {
				return diff, err
			}
			b2, err := os.ReadFile(filepath.Join(pkg2, f))
			if err != nil {
				return diff, err
			}
			if !nonKptfileEquals(string(b1), string(b2)) {
				diff.Insert(f)
			}
		}
	}
	return diff, nil
}

func kptfilesEqual(pkg1, pkg2, filePath string) (bool, error) {
	pkg1Kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, filepath.Join(pkg1, filepath.Dir(filePath)))
	if err != nil {
		return false, err
	}
	pkg2Kf, err := pkg.ReadKptfile(filesys.FileSystemOrOnDisk{}, filepath.Join(pkg2, filepath.Dir(filePath)))
	if err != nil {
		return false, err
	}

	// Diffs in Upstream and UpstreamLock should be ignored.
	pkg1Kf.Upstream = &kptfilev1.Upstream{}
	pkg1Kf.UpstreamLock = &kptfilev1.UpstreamLock{}
	pkg2Kf.Upstream = &kptfilev1.Upstream{}
	pkg2Kf.UpstreamLock = &kptfilev1.UpstreamLock{}

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

// nonKptfileEquals returns true if contents of two non-Kptfiles are equal
// since the changes to addmetricsannotation.CNRMMetricsAnnotation is made
// by kpt, we should not treat it as changes made by user, so delete the annotation
// before comparing
func nonKptfileEquals(s1, s2 string) bool {
	out1 := &bytes.Buffer{}
	out2 := &bytes.Buffer{}
	err := kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(s1)}},
		Filters: []kio.Filter{kio.FilterAll(yaml.AnnotationClearer{Key: attribution.CNRMMetricsAnnotation})},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out1}},
	}.Execute()
	if err != nil {
		return bytes.Equal([]byte(s1), []byte(s2))
	}
	err = kio.Pipeline{
		Inputs:  []kio.Reader{&kio.ByteReader{Reader: bytes.NewBufferString(s2)}},
		Filters: []kio.Filter{kio.FilterAll(yaml.AnnotationClearer{Key: attribution.CNRMMetricsAnnotation})},
		Outputs: []kio.Writer{kio.ByteWriter{Writer: out2}},
	}.Execute()
	if err != nil {
		return bytes.Equal([]byte(s1), []byte(s2))
	}
	return out1.String() == out2.String()
}
