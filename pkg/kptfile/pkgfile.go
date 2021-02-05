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
	"github.com/go-errors/errors"
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

	// OpenAPI contains additional schema for the resources in this package
	// Uses interface{} instead of Node to work around yaml serialization issues
	// See https://github.com/go-yaml/yaml/issues/518 and
	// https://github.com/go-yaml/yaml/issues/575
	OpenAPI interface{} `yaml:"openAPI,omitempty"`

	// Functions contains configuration for running functions
	Functions Functions `yaml:"functions,omitempty"`

	// Parameters for inventory object.
	Inventory *Inventory `yaml:"inventory,omitempty"`

	// rNode is the yaml.RNode representation of Kptfile
	rNode *yaml.RNode `yaml:"inventory,omitempty"`
}

// SetUpstream sets the upstream object in the kptfile
func (kf *KptFile) SetUpstream(upstream *Upstream) error {
	if kf.rNode == nil {
		return errors.Errorf("rNode representation of Kptfile must not be nil")
	}
	upNode, err := kf.rNode.Pipe(yaml.LookupCreate(yaml.MappingNode, UpstreamKey))
	if err != nil {
		return err
	}

	err = setValue(upNode, Type, string(GitOrigin))
	if err != nil {
		return err
	}

	gitNode, err := upNode.Pipe(yaml.LookupCreate(yaml.MappingNode, string(GitOrigin)))
	if err != nil {
		return err
	}

	err = setValue(gitNode, Commit, upstream.Git.Commit)
	if err != nil {
		return err
	}
	err = setValue(gitNode, Repo, upstream.Git.Repo)
	if err != nil {
		return err
	}
	err = setValue(gitNode, Directory, upstream.Git.Directory)
	if err != nil {
		return err
	}
	err = setValue(gitNode, Ref, upstream.Git.Ref)
	if err != nil {
		return err
	}

	if upstream.Stdin.FilenamePattern != "" {
		stdInNode, err := upNode.Pipe(yaml.LookupCreate(yaml.MappingNode, StdIn))
		if err != nil {
			return err
		}

		err = setValue(stdInNode, FilenamePattern, upstream.Stdin.FilenamePattern)
		if err != nil {
			return err
		}
		err = setValue(stdInNode, Original, upstream.Stdin.Original)
		if err != nil {
			return err
		}
	}

	return nil
}

// setValue sets the key-value as child to parent node
func setValue(parent *yaml.RNode, key, value string) error {
	child, err := parent.Pipe(yaml.Lookup(key))
	if child == nil || err != nil {
		err = parent.PipeE(yaml.FieldSetter{
			Name:        key,
			StringValue: value,
		})
		if err != nil {
			return err
		}
	} else {
		if child.Field(key) == nil || child.Field(key).Value == nil {
			return errors.Errorf("error while parsing %s node", key)
		}
		// just update the value if the node already exists
		child.Field(key).Value.YNode().Value = value
	}
	return nil
}

// SetRNode sets RNode in kptfile
func (kf *KptFile) SetRNode(kfNode *yaml.RNode) {
	kf.rNode = kfNode
}

// RNode gets RNode in kptfile
func (kf *KptFile) RNode() *yaml.RNode {
	return kf.rNode
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

// MergeOpenAPI adds the OpenAPI definitions from localKf to updatedKf.
// It takes originalKf as a reference for 3-way merge
// This function is very complex due to serialization issues with yaml.Node.
func (kf *KptFile) MergeOpenAPI(localKf, originalKf KptFile) error {
	if localKf.OpenAPI == nil {
		// no OpenAPI to copy -- do nothing
		return nil
	}
	if kf.OpenAPI == nil {
		// no openAPI at the destination -- just copy it
		kf.OpenAPI = localKf.OpenAPI
		return nil
	}

	// turn the exiting openapi into yaml.Nodes for processing
	// they aren't yaml.Nodes natively due to serialization bugs in the yaml libs
	bUpdated, err := yaml.Marshal(kf.OpenAPI)
	if err != nil {
		return err
	}
	updated, err := yaml.Parse(string(bUpdated))
	if err != nil {
		return err
	}
	bLocal, err := yaml.Marshal(localKf.OpenAPI)
	if err != nil {
		return err
	}
	local, err := yaml.Parse(string(bLocal))
	if err != nil {
		return err
	}

	bOriginal, err := yaml.Marshal(originalKf.OpenAPI)
	if err != nil {
		return err
	}
	original, err := yaml.Parse(string(bOriginal))
	if err != nil {
		return err
	}

	// get the definitions for the source and destination
	updatedDef := updated.Field("definitions")
	if updatedDef == nil {
		// no definitions on the destination, just copy the OpenAPI from the source
		kf.OpenAPI = localKf.OpenAPI
		return nil
	}
	localDef := local.Field("definitions")
	if localDef == nil {
		// no OpenAPI definitions on the source -- do nothings
		return nil
	}
	oriDef := original.Field("definitions")
	if oriDef == nil {
		// no definitions on the destination, fall back to local definitions
		oriDef = localDef
	}

	// merge the definitions
	err = mergeDef(updatedDef, localDef, oriDef)
	if err != nil {
		return err
	}

	// convert the result back to type interface{} and set it on the Kptfile
	s, err := updated.String()
	if err != nil {
		return err
	}
	var newOpenAPI interface{}
	kf.OpenAPI = newOpenAPI
	err = yaml.Unmarshal([]byte(s), &kf.OpenAPI)
	return err
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
	GitOrigin       OriginType = "git"
	StdinOrigin     OriginType = "stdin"
	UpstreamKey                = "upstream"
	Directory                  = "directory"
	Type                       = "type"
	Ref                        = "ref"
	Commit                     = "commit"
	Repo                       = "repo"
	StdIn                      = "stdin"
	FilenamePattern            = "filenamePattern"
	Original                   = "original"
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
