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

// Package pkgfile contains functions for working with KptFile instances.
package pkgfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"lib.kpt.dev/custom"
	"lib.kpt.dev/yaml"
)

// TypeMeta is the TypeMeta for KptFile instances.
var TypeMeta = yaml.ResourceMeta{
	Kind:       "KptFile",
	ApiVersion: "kpt.dev/v1alpha1",
}

// KptFile contains information about a package managed with kpt
type KptFile struct {
	yaml.ResourceMeta `yaml:",inline"`

	// CloneFrom records where the package was originally cloned from
	Upstream Upstream `yaml:"upstream,omitempty"`

	// PackageMeta contains information about the package
	PackageMeta PackageMeta `yaml:"packageMetadata,omitempty"`

	Commands []custom.ResourceCommand `yaml:"commands,omitempty"`
}

type PackageMeta struct {
	// Repo is the location of the package.  e.g. https://github.com/example/com
	Url string `yaml:"url,omitempty"`

	// Email is the email of the package maintainer
	Email string `yaml:"email,omitempty"`

	// License is the package license
	License string `yaml:"license,omitempty"`

	// Version is the package version
	Version string `yaml:"version,omitempty"`

	// Tags can be indexed and are metadata about the package
	Tags []string `yaml:"tags,omitempty"`

	// Man is the path to documentation about the package
	Man string `yaml:"man,omitempty"`

	// ShortDescription contains a short description of the package.
	ShortDescription string `yaml:"shortDescription,omitempty"`
}

// OriginType defines the type of origin for a package
type OriginType string

const (
	// GitOrigin specifies a package as having been cloned from a git repository
	GitOrigin   OriginType = "git"
	StdinOrigin OriginType = "stdin"
)

// Upstream defines where a package was cloned from
type Upstream struct {
	// Type is the type of origin.
	Type OriginType `yaml:"type,omitempty"`

	// Git contains information on the origin of packages cloned from a git repository.
	Git Git `yaml:"git,omitempty"`

	Stdin Stdin `yaml:"stdin,omitempty"`
}

type Stdin struct {
	FilenamePattern string `yaml:"filenamePattern,omitempty"`

	Original string `yaml:"original,omitempty"`
}

// Git contains information on the origin of packages cloned from a git repository.
type Git struct {
	// Commit is the git commit that the package was fetched at
	Commit string `yaml:"commit,omitempty"`

	// Repo is the git repository the package was cloned from.  e.g. https://
	Repo string `yaml:"repo,omitempty"`

	// RepoDirectory is the sub directory of the git repository that the package was cloned from
	Directory string `yaml:"directory,omitempty"`

	// Ref is the git ref the package was cloned from
	Ref string `yaml:"ref,omitempty"`
}

// KptFileName is the name of the KptFile
const KptFileName = "Kptfile"

// ReadFile reads the KptFile in the given directory
func ReadFile(dir string) (KptFile, error) {
	kpgfile := KptFile{ResourceMeta: TypeMeta}
	f, err := os.Open(filepath.Join(dir, KptFileName))
	if err != nil {
		return KptFile{}, fmt.Errorf("unable to read %s: %v", KptFileName, err)
	}
	defer f.Close()

	d := yaml.NewDecoder(f)
	d.KnownFields(true)
	if err = d.Decode(&kpgfile); err != nil {
		return KptFile{}, fmt.Errorf("unable to parse %s: %v", KptFileName, err)
	}
	return kpgfile, nil
}

func WriteFile(dir string, k KptFile) error {
	b, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, KptFileName)); err != nil && !os.IsNotExist(err) {
		return err
	}

	// fyi: perm is ignored if the file already exists
	return ioutil.WriteFile(filepath.Join(dir, KptFileName), b, 0600)
}

// ReadFileStrict reads a Kptfile for a package and validates that it contains required
// Upstream fields.
func ReadFileStrict(pkgPath string) (KptFile, error) {
	kf, err := ReadFile(pkgPath)
	if err != nil {
		return KptFile{}, err
	}

	if kf.Upstream.Type == GitOrigin {
		git := kf.Upstream.Git
		if git.Repo == "" {
			return KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.repo", pkgPath)
		}
		if git.Commit == "" {
			return KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.commit", pkgPath)
		}
		if git.Ref == "" {
			return KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.ref", pkgPath)
		}
		if git.Directory == "" {
			return KptFile{}, fmt.Errorf("%s Kptfile missing upstream.git.directory", pkgPath)
		}
	}
	if kf.Upstream.Type == StdinOrigin {
		stdin := kf.Upstream.Stdin
		if stdin.FilenamePattern == "" {
			return KptFile{}, fmt.Errorf(
				"%s Kptfile missing upstream.stdin.filenamePattern", pkgPath)
		}
		if stdin.Original == "" {
			return KptFile{}, fmt.Errorf(
				"%s Kptfile missing upstream.stdin.original", pkgPath)
		}
	}
	return kf, nil
}
