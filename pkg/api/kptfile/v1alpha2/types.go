// Copyright 2021 Google LLC
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

// Package defines the schema for Kptfile version v1alpha2.
package v1alpha2

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	KptFileName       = "Kptfile"
	KptFileGroup      = "kpt.dev"
	KptFileVersion    = "v1alpha2"
	KptFileAPIVersion = KptFileGroup + "/" + KptFileVersion
)

// KptFile contains information about a package managed with kpt.
type KptFile struct {
	yaml.ResourceMeta `yaml:",inline"`

	// Upstream is a reference to where the package is fetched from.
	Upstream Upstream `yaml:"upstream,omitempty"`

	// PackageMeta contains metadata such as license, documentation, etc.
	PackageMeta PackageMeta `yaml:"packageMeta,omitempty"`

	// Subpackages declares the list of subpackages.
	Subpackages []Subpackage `yaml:"subpackages,omitempty"`

	// Pipeline declares the pipeline of functions.
	Pipeline *Pipeline `yaml:"pipeline,omitempty"`

	// Inventory contains parameters for the inventory object used in apply.
	Inventory *Inventory `yaml:"inventory,omitempty"`
}

// OriginType defines the type of origin for a package
type OriginType string

const (
	// GitOrigin specifies a package as having been cloned from a git repository
	GitOrigin OriginType = "git"
)

// UpdateStrategyType defines the strategy for updating a package from upstream.
type UpdateStrategyType string

const (
	// ResourceMerge performs a structural schema-aware comparison and
	// merges the changes into the local package.
	ResourceMerge UpdateStrategyType = "resource-merge"
	// FastForward fails without updating if the local package was modified
	// since it was fetched.
	FastForward UpdateStrategyType = "fast-forward"
	// ForceDeleteReplace wipes all local changes to the package.
	ForceDeleteReplace UpdateStrategyType = "force-delete-replace"
)

// Upstream is a reference to where the package is fetched from.
type Upstream struct {
	// Type is the type of origin.
	Type OriginType `yaml:"type,omitempty"`

	// Git contains information on the origin of packages fetched from a git repository.
	Git Git `yaml:"git,omitempty"`

	// UpdateStrategy defines how a package is updated from upstream.
	UpdateStrategy UpdateStrategyType `yaml:"updateStrategy,omitempty"`
}

// PackageMeta contains metadata such as license, documentation, etc.
// These fields are not used for any functionality in kpt and are simply passed through.
type PackageMeta struct {
	// URL is the location of the package.  e.g. https://github.com/example/com
	URL string `yaml:"url,omitempty"`

	// Email is the email of the package maintainer
	Email string `yaml:"email,omitempty"`

	// License is the package license
	License string `yaml:"license,omitempty"`

	// Version is a logical package version (ignored by kpt)
	Version string `yaml:"version,omitempty"`

	// Tags enables humans and tools to attach arbitrary package metadata.
	Tags []string `yaml:"tags,omitempty"`

	// Man is the path to documentation about the package
	Man string `yaml:"man,omitempty"`

	// ShortDescription contains a short description of the package.
	ShortDescription string `yaml:"shortDescription,omitempty"`
}

// Subpackages declares a local or remote subpackage.
type Subpackage struct {
	// Name of the immediate subdirectory relative to this Kptfile where the suppackage
	// either exists (local subpackages) or will be fetched to (remote subpckages).
	// This must be unique across all subpckages of a package.
	LocalDir string `yaml:"localDir,omitempty"`
	// Whether a subpackage is local or remote is determined by whether Upstream is specified.
	// Upstream is a reference to where the package is fetched from.
	Upstream Upstream `yaml:"upstream,omitempty"`
}

// Git contains information on the origin of packages fetched from a git repository.
type Git struct {
	// Commit is the git commit that the package was fetched at
	Commit string `yaml:"commit,omitempty"`

	// Repo is the git repository the package was cloned from.  e.g. https://
	Repo string `yaml:"repo,omitempty"`

	// Directory is the sub directory of the git repository that the package was cloned from
	Directory string `yaml:"directory,omitempty"`

	// Ref is the git ref the package was cloned from
	Ref string `yaml:"ref,omitempty"`
}

// Pipeline declares a pipeline of functions used to mutate or validate resources.
type Pipeline struct {
	//  Sources defines the source packages to resolve as input to the pipeline. Possible values:
	//  a) A slash-separated, OS-agnostic relative package path which may include '.' and '..' e.g. './base', '../foo'
	//     The source package is resolved recursively.
	//  b) Resources in this package using '.'. Meta resources such as the Kptfile, Pipeline, and function configs
	//     are excluded.
	//  c) Resources in this package AND all resolved subpackages using './*'
	//
	// Resultant list of resources are ordered:
	// - According to the order of sources specified in this array.
	// - When using './*': Subpackages are resolved in alphanumerical order before package resources.
	//
	// When omitted, defaults to './*'.
	// Sources []string `yaml:"sources,omitempty"`

	// Following fields define the sequence of functions in the pipeline.
	// Input of the first function is the resolved sources.
	// Input of the second function is the output of the first function, and so on.
	// Order of operation: mutators, validators

	// Mutators defines a list of of KRM functions that mutate resources.
	Mutators []Function `yaml:"mutators,omitempty"`

	// Validators defines a list of KRM functions that validate resources.
	// Validators are not permitted to mutate resources.
	Validators []Function `yaml:"validators,omitempty"`
}

// Function specifies a KRM function.
type Function struct {
	// `Image` specifies the function container image.
	// It can either be fully qualified, e.g.:
	//
	//	image: gcr.io/kpt-fn/set-label
	//
	// Optionally, kpt can be configured to use a image
	// registry host-path that will be used to resolve the image path in case
	// the image path is missing (Defaults to gcr.io/kpt-fn).
	// e.g. The following resolves to gcr.io/kpt-fn/set-label:
	//
	//	image: set-label
	Image string `yaml:"image,omitempty"`

	// `Config` specifies an inline k8s resource used as the function config.
	// Config, ConfigPath, and ConfigMap fields are mutually exclusive.
	Config yaml.Node `yaml:"config,omitempty"`

	// `ConfigPath` specifies a relative path to a file in the current directory
	// containing a K8S resource used as the function config. This resource is
	// excluded when resolving 'sources', and as a result cannot be operated on
	// by the pipeline.
	ConfigPath string `yaml:"configPath,omitempty"`

	// `ConfigMap` is a convenient way to specify a function config of kind ConfigMap.
	ConfigMap map[string]string `yaml:"configMap,omitempty"`
}

// Inventory encapsulates the parameters for the inventory object. All of the
// the parameters are required if any are set.
type Inventory struct {
	Namespace string `yaml:"namespace,omitempty"`
	Name      string `yaml:"name,omitempty"`
	// Unique label to identify inventory object in cluster.
	InventoryID string            `yaml:"inventoryID,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}
