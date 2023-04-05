// Copyright 2021 The kpt Authors
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

package v1

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoogleContainerTools/kpt/internal/types"
	"sigs.k8s.io/kustomize/api/konfig"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// constants related to kustomize
	kustomizationAPIGroup = "kustomize.config.k8s.io"
)

func (kf *KptFile) Validate(fsys filesys.FileSystem, pkgPath types.UniquePath) error {
	if err := kf.Pipeline.validate(fsys, pkgPath); err != nil {
		return fmt.Errorf("invalid pipeline: %w", err)
	}
	// TODO: validate other fields
	return nil
}

// validate will validate all fields in the Pipeline
// 'mutators' and 'validators' share same schema and
// they are valid if all functions in them are ALL valid.
func (p *Pipeline) validate(fsys filesys.FileSystem, pkgPath types.UniquePath) error {
	if p == nil {
		return nil
	}
	for i := range p.Mutators {
		f := p.Mutators[i]
		err := f.validate(fsys, "mutators", i, pkgPath)
		if err != nil {
			return fmt.Errorf("function %q: %w", f.Image, err)
		}
	}
	for i := range p.Validators {
		f := p.Validators[i]
		err := f.validate(fsys, "validators", i, pkgPath)
		if err != nil {
			return fmt.Errorf("function %q: %w", f.Image, err)
		}
	}
	return nil
}

func (f *Function) validate(fsys filesys.FileSystem, fnType string, idx int, pkgPath types.UniquePath) error {
	if f.Image == "" && f.Exec == "" {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d]", fnType, idx),
			Reason: "must specify a functon (`image` or `exec`) to execute",
		}
	}
	if f.Image != "" && f.Exec != "" {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d]", fnType, idx),
			Reason: "must not specify both `image` and `exec` at the same time",
		}
	}
	if f.Image != "" {
		err := ValidateFunctionImageURL(f.Image)
		if err != nil {
			return &ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].image", fnType, idx),
				Value:  f.Image,
				Reason: err.Error(),
			}
		}
	}
	// TODO(droot): validate the exec

	if len(f.ConfigMap) != 0 && f.ConfigPath != "" {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d]", fnType, idx),
			Reason: "functionConfig must not specify both `configMap` and `configPath` at the same time",
		}
	}

	if f.ConfigPath != "" {
		if err := validateFnConfigPathSyntax(f.ConfigPath); err != nil {
			return &ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].configPath", fnType, idx),
				Value:  f.ConfigPath,
				Reason: err.Error(),
			}
		}
		if _, err := GetValidatedFnConfigFromPath(fsys, pkgPath, f.ConfigPath); err != nil {
			return &ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].configPath", fnType, idx),
				Value:  f.ConfigPath,
				Reason: err.Error(),
			}
		}
	}
	return nil
}

// ValidateFunctionImageURL validates the function name.
// According to Docker implementation
// https://github.com/docker/distribution/blob/master/reference/reference.go. A valid
// name definition is:
//
//	name                            := [domain '/'] path-component ['/' path-component]*
//	domain                          := domain-component ['.' domain-component]* [':' port-number]
//	domain-component                := /([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])/
//	port-number                     := /[0-9]+/
//	path-component                  := alpha-numeric [separator alpha-numeric]*
//	alpha-numeric                   := /[a-z0-9]+/
//	separator                       := /[_.]|__|[-]*/
func ValidateFunctionImageURL(name string) error {
	pathComponentRegexp := `(?:[a-z0-9](?:(?:[_.]|__|[-]*)[a-z0-9]+)*)`
	domainComponentRegexp := `(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])`
	domainRegexp := fmt.Sprintf(`%s(?:\.%s)*(?:\:[0-9]+)?`, domainComponentRegexp, domainComponentRegexp)
	nameRegexp := fmt.Sprintf(`(?:%s\/)?%s(?:\/%s)*`, domainRegexp,
		pathComponentRegexp, pathComponentRegexp)
	tagRegexp := `(?:[\w][\w.-]{0,127})`
	shaRegexp := `(sha256:[a-zA-Z0-9]{64})`
	versionRegexp := fmt.Sprintf(`(%s|%s)`, tagRegexp, shaRegexp)
	r := fmt.Sprintf(`^(?:%s(?:(\:|@)%s)?)$`, nameRegexp, versionRegexp)

	matched, err := regexp.MatchString(r, name)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("function name %q is invalid", name)
	}
	return nil
}

// validateFnConfigPathSyntax validates syntactic correctness of given functionConfig path
// and return an error if it's invalid.
func validateFnConfigPathSyntax(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("path must not be empty")
	}
	p = filepath.Clean(p)
	if filepath.IsAbs(p) {
		return fmt.Errorf("path must be relative")
	}
	if strings.Contains(p, "..") {
		// fn config must not live outside the package directory
		// Allowing outside path opens up an attack vector that allows
		// reading any YAML file on package consumer's machine.
		return fmt.Errorf("path must not be outside the package")
	}
	return nil
}

// GetValidatedFnConfigFromPath validates the functionConfig at the path specified by
// the package path (pkgPath) and configPath, returning the functionConfig as an
// RNode if the validation is successful.
func GetValidatedFnConfigFromPath(fsys filesys.FileSystem, pkgPath types.UniquePath, configPath string) (*yaml.RNode, error) {
	path := filepath.Join(string(pkgPath), configPath)
	file, err := fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("functionConfig must exist in the current package")
	}
	defer file.Close()
	reader := kio.ByteReader{Reader: file, PreserveSeqIndent: true, WrapBareSeqNode: true, DisableUnwrapping: true}
	nodes, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read functionConfig %q: %w", configPath, err)
	}
	if len(nodes) > 1 {
		return nil, fmt.Errorf("functionConfig %q must not contain more than one config, got %d", configPath, len(nodes))
	}
	if err := IsKRM(nodes[0]); err != nil {
		return nil, fmt.Errorf("functionConfig %q: %s", configPath, err.Error())
	}
	return nodes[0], nil
}

// AreKRM validates if given resources are valid KRM resources.
func AreKRM(nodes []*yaml.RNode) error {
	for i := range nodes {
		if err := IsKRM(nodes[i]); err != nil {
			path, _, _ := kioutil.GetFileAnnotations(nodes[i])
			return fmt.Errorf("%s: %s", path, err.Error())
		}
	}
	return nil
}

// IsKRM validates if given resource is a valid KRM resource by ensuring
// that resource has a valid apiVersion, kind and metadata.name field.
// It excludes kustomization resource from KRM check.
func IsKRM(n *yaml.RNode) error {
	if isKustomization(n) {
		// exclude kustomization files from KRM check
		// https://github.com/GoogleContainerTools/kpt/issues/2388
		return nil
	}
	meta, err := n.GetMeta()
	if err != nil {
		return fmt.Errorf("resource must have `apiVersion`, `kind`, and `name`")
	}
	if meta.APIVersion == "" {
		return fmt.Errorf("resource must have `apiVersion`")
	}
	if meta.Kind == "" {
		return fmt.Errorf("resource must have `kind`")
	}
	if meta.Name == "" {
		return fmt.Errorf("resource must have `metadata.name`")
	}
	return nil
}

// isKustomization determines if given YAML is a kustomization file or resource.
func isKustomization(n *yaml.RNode) bool {
	resourcePath, _, err := kioutil.GetFileAnnotations(n)
	if err == nil {
		// perform the check only if we are able to reliably
		// read the file path of the resource
		resourceFile := filepath.Base(resourcePath)

		for _, kustomizationFileName := range konfig.RecognizedKustomizationFileNames() {
			if resourceFile == kustomizationFileName {
				return true
			}
		}
	}
	meta, err := n.GetMeta()
	if err != nil {
		return false
	}

	if strings.HasPrefix(meta.APIVersion, kustomizationAPIGroup) {
		return true
	}

	if meta.APIVersion == "" && meta.Kind == kustomizetypes.KustomizationKind {
		return true
	}

	return false
}

// ValidateError is the error returned when validation fails.
type ValidateError struct {
	// Field is the field that causes error
	Field string
	// Value is the value of invalid field
	Value string
	// Reason is the reason for the error
	Reason string
}

func (e *ValidateError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Kptfile is invalid:\nField: `%s`\n", e.Field))
	if e.Value != "" {
		sb.WriteString(fmt.Sprintf("Value: %q\n", e.Value))
	}
	sb.WriteString(fmt.Sprintf("Reason: %s\n", e.Reason))
	return sb.String()
}
