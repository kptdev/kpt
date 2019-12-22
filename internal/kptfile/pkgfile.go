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

	// Substitutions are substitutions that may be performed against the package
	Substitutions []Substitution `yaml:"substitutions,omitempty"`
}

// Substitution defines how to substitute a value into a package
type Substitution struct {
	// Marker is the string Marker to be substituted
	Marker string `yaml:"marker,omitempty"`

	// Marker is the string Marker to be substituted
	Required *bool `yaml:"required,omitempty"`

	// Paths are the search paths to look for the Marker in each Resource
	Paths []Path `yaml:"paths,omitempty"`

	// InputParameter defines the input value to substitute
	InputParameter `yaml:",inline,omitempty"`
}

// Path defines a path to a field
type Path struct {
	Path []string `yaml:"path,omitempty"`
}

// InputType defines the type of input to register
type InputType string

const (
	// String defines a string flag
	String InputType = "string"
	// Bool defines a bool flag
	Bool = "bool"
	// Float defines a float flag
	Float = "float"
	// Int defines an int flag
	Int = "int"
)

func (it InputType) Tag() string {
	switch it {
	case String:
		return "!!str"
	case Bool:
		return "!!bool"
	case Int:
		return "!!int"
	case Float:
		return "!!float"
	}
	return ""
}

// InputParameter defines an input parameter that should be registered with the templates.
type InputParameter struct {
	// Type is the type of the input
	Type InputType `yaml:"type"`

	// Description is the description of the input value
	Description string `yaml:"description"`

	// Name is the name of the input
	Name string `yaml:"name"`

	// StringValue is the value to substitute in
	// +optional
	StringValue string `yaml:"stringValue"`
}

type Dependency struct {
	Name            string `yaml:"name,omitempty"`
	Upstream        `yaml:",inline,omitempty"`
	Path            string `yaml:"path,omitempty"`
	EnsureNotExists bool   `yaml:"ensureNotExists,omitempty"`
	Strategy        string `yaml:"updateStrategy,omitempty"`
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

	Functions []*yaml.Node `yaml:"functions,omitempty"`
}
