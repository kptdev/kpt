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
package kptfile

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// KptFileName is the name of the KptFile
const KptFileName = "Kptfile"

// TypeMeta is the TypeMeta for KptFile instances.
var TypeMeta = yaml.ResourceMeta{
	Kind:       KptFileName,
	APIVersion: "kpt.dev/v1alpha1",
}

// KptFile contains information about a package managed with kpt
type KptFile struct {
	yaml.ResourceMeta `yaml:",inline"`

	// CloneFrom records where the package was originally cloned from
	Upstream Upstream `yaml:"upstream,omitempty"`

	// PackageMeta contains information about the package
	PackageMeta PackageMeta `yaml:"packageMetadata,omitempty"`

	Dependencies []Dependency `yaml:"dependencies,omitempty"`

	// OpenAPI contains additional schema for the resources in this package
	// Uses interface{} instead of Node to work around yaml serialization issues
	// See https://github.com/go-yaml/yaml/issues/518 and
	// https://github.com/go-yaml/yaml/issues/575
	OpenAPI interface{} `yaml:"openAPI,omitempty"`

	// Functions contains configuration for running functions
	Functions Functions `yaml:"functions,omitempty"`
}

type Functions struct {
	// AutoRunStarlark will cause starlark functions to automatically be run.
	AutoRunStarlark bool `yaml:"autoRunStarlark,omitempty"`

	// StarlarkFunctions is a list of starlark functions to run
	StarlarkFunctions []StarlarkFunction `yaml:"starlarkFunctions,omitempty"`
}

type StarlarkFunction struct {
	// Name is the name that will be given to the program
	Name string `yaml:"name,omitempty"`
	// Path is the path to the *.star script to run
	Path string `yaml:"path,omitempty"`
}

// MergeOpenAPI adds the OpenAPI definitions from file to k.
// This function is very complex due to serialization issues with yaml.Node.
func (k *KptFile) MergeOpenAPI(file KptFile) error {
	if file.OpenAPI == nil {
		// no OpenAPI to copy -- do nothing
		return nil
	}
	if k.OpenAPI == nil {
		// no openAPI at the destination -- just copy it
		k.OpenAPI = file.OpenAPI
		return nil
	}

	// turn the exiting openapi into yaml.Nodes for processing
	// they aren't yaml.Nodes natively due to serialization bugs in the yaml libs
	bTo, err := yaml.Marshal(k.OpenAPI)
	if err != nil {
		return err
	}
	to, err := yaml.Parse(string(bTo))
	if err != nil {
		return err
	}
	bFrom, err := yaml.Marshal(file.OpenAPI)
	if err != nil {
		return err
	}
	from, err := yaml.Parse(string(bFrom))
	if err != nil {
		return err
	}

	// get the definitions for the source and destination
	toDef := to.Field("definitions")
	if toDef == nil {
		// no definitions on the destination, just copy the OpenAPI from the source
		k.OpenAPI = file.OpenAPI
		return nil
	}
	fromDef := from.Field("definitions")
	if fromDef == nil {
		// OpenAPI definitions on the source -- do nothings
		return nil
	}

	err = fromDef.Value.VisitFields(func(node *yaml.MapNode) error {
		// copy each definition from the source to the destination
		return toDef.Value.PipeE(yaml.FieldSetter{
			Name:  node.Key.YNode().Value,
			Value: node.Value})
	})
	if err != nil {
		return err
	}

	// convert the result back to type interface{} and set it on the Kptfile
	s, err := to.String()
	if err != nil {
		return err
	}
	var newOpenAPI interface{}
	k.OpenAPI = newOpenAPI
	err = yaml.Unmarshal([]byte(s), &k.OpenAPI)
	return err
}

type Dependency struct {
	Name            string `yaml:"name,omitempty"`
	Upstream        `yaml:",inline,omitempty"`
	EnsureNotExists bool       `yaml:"ensureNotExists,omitempty"`
	Strategy        string     `yaml:"updateStrategy,omitempty"`
	Functions       []Function `yaml:"functions,omitempty"`
	AutoSet         bool       `yaml:"autoSet,omitempty"`
}

type PackageMeta struct {
	// URL is the location of the package.  e.g. https://github.com/example/com
	URL string `yaml:"url,omitempty"`

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

type Function struct {
	Config yaml.Node `yaml:"config,omitempty"`
	Image  string    `yaml:"image,omitempty"`
}
