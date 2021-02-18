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
	"fmt"
	"reflect"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// KptFileName is the name of the KptFile
const (
	KptFileName       = "Kptfile"
	KptFileGroup      = "kpt.dev"
	KptFileVersion    = "v1alpha1"
	KptFileAPIVersion = KptFileGroup + "/" + KptFileVersion
)

// TypeMeta is the TypeMeta for KptFile instances.
var TypeMeta = yaml.ResourceMeta{
	TypeMeta: yaml.TypeMeta{
		APIVersion: KptFileAPIVersion,
		Kind:       KptFileName,
	},
}

// KptFile contains information about a package managed with kpt
type KptFile struct {
	yaml.ResourceMeta `yaml:",inline"`

	// CloneFrom records where the package was originally cloned from
	Upstream Upstream `yaml:"upstream,omitempty"`

	// PackageMeta contains information about the package
	PackageMeta PackageMeta `yaml:"packageMetadata,omitempty"`

	Dependencies []Dependency `yaml:"dependencies,omitempty"`

	Subpackages []Subpackage `yaml:"subpackages,omitempty"`

	// OpenAPI contains additional schema for the resources in this package
	// Uses interface{} instead of Node to work around yaml serialization issues
	// See https://github.com/go-yaml/yaml/issues/518 and
	// https://github.com/go-yaml/yaml/issues/575
	OpenAPI interface{} `yaml:"openAPI,omitempty"`

	// Parameters for inventory object.
	Inventory *Inventory `yaml:"inventory,omitempty"`
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

// MergeOpenAPI takes the openAPI information from local, updated and original
// and does a 3-way merge. It doesn't change any of the parameters, but returns
// a new data structure with the merge openapi information.
// This function is very complex due to serialization issues with yaml.Node.
func MergeOpenAPI(localOA, updatedOA, originalOA interface{}) (interface{}, error) {
	// toRNode turns a data structure containing openAPI information into
	// a RNode reference.
	toRNode := func(s interface{}) (*yaml.RNode, error) {
		b, err := yaml.Marshal(s)
		if err != nil {
			return nil, err
		}
		return yaml.Parse(string(b))
	}

	// clone makes a new copy of the data structure by marshalling it to
	// yaml and then unmarshalling into a different object.
	clone := func(s interface{}) (interface{}, error) {
		b, err := yaml.Marshal(s)
		if err != nil {
			return nil, err
		}
		var i interface{}
		err = yaml.Unmarshal(b, &i)
		return i, err
	}

	if localOA == nil {
		return clone(updatedOA)
	}

	if updatedOA == nil {
		return clone(localOA)
	}

	// turn the exiting openapi into yaml.Nodes for processing
	// they aren't yaml.Nodes natively due to serialization bugs in the yaml libs
	local, err := toRNode(localOA)
	if err != nil {
		return nil, err
	}
	updated, err := toRNode(updatedOA)
	if err != nil {
		return nil, err
	}
	original, err := toRNode(originalOA)
	if err != nil {
		return nil, err
	}

	// get the definitions for the source and destination
	updatedDef := updated.Field("definitions")
	if updatedDef == nil {
		// no definitions from updated, just return the openapi defs from
		// local
		return clone(localOA)
	}
	localDef := local.Field("definitions")
	if localDef == nil {
		// no openapi defs on local. Just return the openapi defs from
		// updated.
		return clone(updatedOA)
	}
	oriDef := original.Field("definitions")
	if oriDef == nil {
		// no definitions on the destination, fall back to local definitions
		oriDef = localDef
	}

	// merge the definitions
	err = mergeDef(updatedDef, localDef, oriDef)
	if err != nil {
		return nil, err
	}

	// convert the result back to type interface{} and set it on the Kptfile
	s, err := updated.String()
	if err != nil {
		return nil, err
	}
	var newOpenAPI interface{}
	err = yaml.Unmarshal([]byte(s), &newOpenAPI)
	return newOpenAPI, err
}

// mergeDef takes localDef, originalDef and updateDef, it iterates through the unique keys of localDef
// and updateDef, skip copy the local node if nothing changed or updateDef get deleted.
// It deletes the node from updateDef if node get deleted in localDef
func mergeDef(updatedDef, localDef, originalDef *yaml.MapNode) error {
	localKeys, err := localDef.Value.Fields()
	if err != nil {
		return err
	}
	updatedKeys, err := updatedDef.Value.Fields()
	if err != nil {
		return nil
	}
	keys := append(updatedKeys, localKeys...)

	unique := make(map[string]bool, len(keys))
	for _, key := range keys {
		if unique[key] {
			continue
		}
		unique[key] = true

		node := localDef.Value.Field(key)
		if node == nil {
			node = updatedDef.Value.Field(key)
		}

		if shouldSkipCopy(updatedDef, localDef, originalDef, key) {
			continue
		}

		if shouldRemoveValue(updatedDef, localDef, originalDef, key) {
			err = updatedDef.Value.PipeE(yaml.FieldClearer{Name: key})
			if err != nil {
				return err
			}
			continue
		}

		err = updatedDef.Value.PipeE(yaml.FieldSetter{
			Name:  key,
			Value: node.Value})
		if err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipCopy decides if a node with key should be copied from fromDef to toDef
func shouldSkipCopy(updatedDef, localDef, originalDef *yaml.MapNode, key string) bool {
	if originalDef == nil || updatedDef == nil || localDef == nil {
		return false
	}
	localVal := localDef.Value.Field(key)
	originalVal := originalDef.Value.Field(key)
	updatedVal := updatedDef.Value.Field(key)
	if localVal == nil || originalVal == nil {
		return false
	}

	localValStr, err := localVal.Value.String()
	if err != nil {
		return false
	}
	originalValStr, err := originalVal.Value.String()
	if err != nil {
		return false
	}

	// skip copying if the definition is deleted from upstream
	if updatedVal == nil {
		return true
	}
	// skip copying if original val matches with from val(local val)
	return localValStr == originalValStr
}

// shouldRemoveValue decides if a node with key should be removed from Def
func shouldRemoveValue(updatedDef, localDef, originalDef *yaml.MapNode, key string) bool {
	localVal := localDef.Value.Field(key)
	originalVal := originalDef.Value.Field(key)
	updatedVal := updatedDef.Value.Field(key)

	if originalVal == nil || updatedVal == nil {
		return false
	}

	originalValStr, err := originalVal.Value.String()
	if err != nil {
		return false
	}

	updatedValStr, err := updatedVal.Value.String()
	if err != nil {
		return false
	}

	if localVal == nil && originalValStr == updatedValStr {
		return true
	}

	return false
}

// MergeSubpackages takes the subpackage information from local, updated
// and original and does a 3-way merge. The result is returned as a new slice.
// The passed in data structures are not changed.
func MergeSubpackages(local, updated, original []Subpackage) ([]Subpackage, error) {
	// find is a helper function that returns a subpackage with the provided
	// key from the slice.
	find := func(key string, slice []Subpackage) (Subpackage, bool) {
		for i := range slice {
			sp := slice[i]
			if sp.LocalDir == key {
				return sp, true
			}
		}
		return Subpackage{}, false
	}

	// Create a new slice to contain the merged result.
	var merged []Subpackage

	// Create a slice that contains all keys available from both updated
	// and local. We add keys from updated first, so subpackages added
	// locally will be at the end of the slice after merge.
	var dirKeys []string
	for _, sp := range updated {
		dirKeys = append(dirKeys, sp.LocalDir)
	}
	for _, sp := range local {
		dirKeys = append(dirKeys, sp.LocalDir)
	}

	// The slice of keys might contain duplicates, so keep track of which
	// keys we have seen.
	seen := make(map[string]bool)
	for _, key := range dirKeys {
		// Skip subpackages that we have already merged.
		if seen[key] {
			continue
		}
		seen[key] = true

		// Look up the package with the given name from all three sources.
		l, lok := find(key, local)
		u, uok := find(key, updated)
		o, ook := find(key, original)

		// If we find a remote subpackage defined in both local and updated, but
		// not in the original, it must have been added both in local and updated.
		// This is an error and the user must resolve this.
		if !ook && uok && lok {
			return merged, fmt.Errorf("remote subpackage with localDir %s added in both local and upstream", key)
		}

		// If not in either upstream or local, we don't need to add it.
		if !lok && !uok {
			continue
		}

		// If deleted from upstream, we only remove it if local is unchanged.
		if ook && !uok {
			if !reflect.DeepEqual(o, l) {
				merged = append(merged, l)
			}
			continue
		}

		// If deleted from local, we don't want to re-add it from upstream.
		if ook && !lok {
			continue
		}

		// If key not found in local, we use the version from updated.
		if !lok {
			merged = append(merged, u)
			continue
		}
		// If key not found in updated, we use the version from local.
		if !uok {
			merged = append(merged, l)
			continue
		}

		// If we changes to local compared with original, we keep the local
		// version. Otherwise, we take hte version from updated.
		if reflect.DeepEqual(o, l) {
			merged = append(merged, u)
		} else {
			merged = append(merged, l)
		}
	}
	return merged, nil
}

type Dependency struct {
	Name            string `yaml:"name,omitempty"`
	Upstream        `yaml:",inline,omitempty"`
	EnsureNotExists bool   `yaml:"ensureNotExists,omitempty"`
	Strategy        string `yaml:"updateStrategy,omitempty"`
	AutoSet         bool   `yaml:"autoSet,omitempty"`
}

type Subpackage struct {
	LocalDir       string `yaml:"localDir,omitempty"`
	Git            Git    `yaml:"git,omitempty"`
	UpdateStrategy string `yaml:"updateStrategy,omitempty"`
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
