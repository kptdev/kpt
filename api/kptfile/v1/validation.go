// Copyright 2021,2026 The kpt Authors
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
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/distribution/reference"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// constants related to kustomize
	kustomizationAPIGroup = "kustomize.config.k8s.io"
	kustomizationKind     = "Kustomization"
)

func (kf *KptFile) Validate(fsys filesys.FileSystem, pkgPath UniquePath) error {
	if err := kf.Pipeline.validate(fsys, pkgPath); err != nil {
		return fmt.Errorf("invalid pipeline: %w", err)
	}
	if err := kf.Upstream.validate(); err != nil {
		return fmt.Errorf("invalid upstream: %w", err)
	}
	if err := kf.UpstreamLock.validate(); err != nil {
		return fmt.Errorf("invalid upstreamLock: %w", err)
	}
	if err := kf.Info.validate(); err != nil {
		return fmt.Errorf("invalid info: %w", err)
	}
	if kf.Inventory != nil {
		if err := kf.Inventory.validate(); err != nil {
			return fmt.Errorf("invalid inventory: %w", err)
		}
	}
	return nil
}

// validate checks the Upstream fields for consistency.
func (u *Upstream) validate() error {
	if u == nil {
		return nil
	}
	if u.Type != "" && u.Type != GitOrigin {
		return &ValidateError{Field: "upstream.type", Value: string(u.Type), Reason: fmt.Sprintf("must be %q when set", GitOrigin)}
	}
	if u.Type == GitOrigin && u.Git == nil {
		return &ValidateError{Field: "upstream.git", Reason: `must be set when upstream.type is "git"`}
	}
	if u.Git != nil {
		if u.Git.Repo == "" {
			return &ValidateError{Field: "upstream.git.repo", Reason: "must not be empty"}
		} else if err := validateGitRepo(u.Git.Repo); err != nil {
			return &ValidateError{Field: "upstream.git.repo", Value: u.Git.Repo, Reason: err.Error()}
		}
		if u.Git.Ref == "" {
			return &ValidateError{Field: "upstream.git.ref", Reason: "must not be empty"}
		}
	}
	if u.UpdateStrategy != "" {
		if _, err := ToUpdateStrategy(string(u.UpdateStrategy)); err != nil {
			return &ValidateError{Field: "upstream.updateStrategy", Value: string(u.UpdateStrategy), Reason: err.Error()}
		}
	}
	return nil
}

// validateGitRepo accepts anything git can clone from: a URL with a scheme
// (https://, ssh://, file://, ...), an scp-style remote ([user@]host:path),
// or a local path.
func validateGitRepo(repo string) error {
	if strings.ContainsAny(repo, " 	\r\n") {
		return fmt.Errorf("must not contain whitespace")
	}
	if strings.Contains(repo, "://") {
		if _, err := url.Parse(repo); err != nil {
			return fmt.Errorf("invalid URL: %s", err)
		}
	}
	return nil
}

// validate checks the Locator (upstreamLock) fields for consistency.
func (l *Locator) validate() error {
	if l == nil {
		return nil
	}
	if l.Type != "" && l.Type != GitOrigin && l.Type != GenericOrigin {
		return &ValidateError{Field: "upstreamLock.type", Value: string(l.Type),
			Reason: fmt.Sprintf("must be %q or %q when set", GitOrigin, GenericOrigin)}
	}
	if l.Git != nil && l.Generic != nil {
		return &ValidateError{Field: "upstreamLock", Reason: "must not specify both `git` and `generic`"}
	}
	if l.Type == GitOrigin && l.Git == nil {
		return &ValidateError{Field: "upstreamLock.git", Reason: `must be set when type is "git"`}
	}
	if l.Type == GenericOrigin && l.Generic == nil {
		return &ValidateError{Field: "upstreamLock.generic", Reason: `must be set when type is "generic"`}
	}
	if l.Git != nil {
		if l.Git.Repo == "" {
			return &ValidateError{Field: "upstreamLock.git.repo", Reason: "must not be empty"}
		}
		if l.Git.Ref == "" {
			return &ValidateError{Field: "upstreamLock.git.ref", Reason: "must not be empty"}
		}
		if l.Git.Commit == "" {
			return &ValidateError{Field: "upstreamLock.git.commit", Reason: "must not be empty"}
		}
	}
	return nil
}

// validate checks the PackageInfo fields.
func (info *PackageInfo) validate() error {
	if info == nil {
		return nil
	}
	for i, rg := range info.ReadinessGates {
		if rg.ConditionType == "" {
			return &ValidateError{Field: fmt.Sprintf("info.readinessGates[%d].conditionType", i), Reason: "must not be empty"}
		}
	}
	if info.LicenseFile != "" {
		p := filepath.Clean(info.LicenseFile)
		if filepath.IsAbs(p) {
			return &ValidateError{Field: "info.licenseFile", Value: info.LicenseFile, Reason: "must be a relative path"}
		}
		if strings.Contains(p, "..") {
			return &ValidateError{Field: "info.licenseFile", Value: info.LicenseFile, Reason: "must not reference a path outside the package"}
		}
	}
	return nil
}

// validate checks that inventory fields are all-or-nothing.
func (inv *Inventory) validate() error {
	hasName, hasNS, hasID := inv.Name != "", inv.Namespace != "", inv.InventoryID != ""
	if hasName == hasNS && hasNS == hasID {
		return nil // all set or all empty — both valid
	}
	var missing []string
	if !hasName {
		missing = append(missing, "`name`")
	}
	if !hasNS {
		missing = append(missing, "`namespace`")
	}
	if !hasID {
		missing = append(missing, "`inventoryID`")
	}
	return &ValidateError{Field: "inventory", Reason: fmt.Sprintf(
		"all of `name`, `namespace`, and `inventoryID` must be specified; missing: %s",
		strings.Join(missing, ", "))}
}

// validate will validate all fields in the Pipeline
// 'mutators' and 'validators' share same schema and
// they are valid if all functions in them are ALL valid.
func (p *Pipeline) validate(fsys filesys.FileSystem, pkgPath UniquePath) error {
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

func (f *Function) validate(fsys filesys.FileSystem, fnType string, idx int, pkgPath UniquePath) error {
	if err := f.validateExecutor(fnType, idx); err != nil {
		return err
	}
	if err := f.validateConfigSources(fnType, idx); err != nil {
		return err
	}
	if f.ConfigRef != nil {
		if err := f.ConfigRef.validate(fnType, idx); err != nil {
			return err
		}
	}
	if f.ConfigPath != "" {
		if err := f.validateConfigPath(fsys, fnType, idx, pkgPath); err != nil {
			return err
		}
	}
	if f.Selectors != nil {
		for i, s := range f.Selectors {
			if err := s.validate(fnType, idx, "selectors", i); err != nil {
				return err
			}
		}
	}
	if f.Exclusions != nil {
		for i, e := range f.Exclusions {
			if err := e.validate(fnType, idx, "exclude", i); err != nil {
				return err
			}
		}
	}
	if err := f.validateExecutor(fnType, idx); err != nil {
		return err
	}
	return nil
}

// validateExecutor checks that exactly one of Image or Exec is specified and that
// the image reference is syntactically valid.
func (f *Function) validateExecutor(fnType string, idx int) error {
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
		if err := ValidateFunctionImageURL(f.Image); err != nil {
			return &ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].image", fnType, idx),
				Value:  f.Image,
				Reason: err.Error(),
			}
		}
	}
	// TODO(droot): validate the exec
	return nil
}

// validateConfigSources ensures at most one of configMap, configPath, or configRef is set.
func (f *Function) validateConfigSources(fnType string, idx int) error {
	configSources := 0
	if len(f.ConfigMap) != 0 {
		configSources++
	}
	if f.ConfigPath != "" {
		configSources++
	}
	if f.ConfigRef != nil {
		configSources++
	}
	if configSources > 1 {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d]", fnType, idx),
			Reason: "functionConfig must specify at most one of `configMap`, `configPath`, or `configRef`",
		}
	}
	return nil
}

// validateConfigPath validates the configPath syntax and verifies the referenced file exists.
func (f *Function) validateConfigPath(fsys filesys.FileSystem, fnType string, idx int, pkgPath UniquePath) error {
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
	return nil
}

// ValidateFunctionImageURL validates the function image reference.
//
// The reference must conform to the standard OCI/Docker reference grammar
// implemented by github.com/distribution/reference:
//
//	reference := name [ ":" tag ] [ "@" digest ]
//
// This means an image may carry a tag
// (e.g. ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.5),
// a digest (e.g. ghcr.io/kptdev/krm-functions-catalog/set-namespace@sha256:...),
// or both at the same time
// (e.g. ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.5@sha256:...).
func ValidateFunctionImageURL(name string) error {
	if _, err := reference.Parse(name); err != nil {
		return fmt.Errorf("function image reference %q is invalid: %w", name, err)
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

// validate checks that the Selector fields are consistent.
func (s Selector) validate(fnType string, idx int, selectorType string, selectorIdx int) error {
	if s.ResourceFileRegexp != "" {
		if _, err := regexp.Compile(s.ResourceFileRegexp); err != nil {
			return &ValidateError{
				Field:  fmt.Sprintf("pipeline.%s[%d].%s[%d].resourceFileRegexp", fnType, idx, selectorType, selectorIdx),
				Value:  s.ResourceFileRegexp,
				Reason: fmt.Sprintf("invalid regular expression: %s", err),
			}
		}
	}
	return nil
}

// validate checks that the ResourceReference has the required fields set.
func (r *ResourceReference) validate(fnType string, idx int) error {
	if r.Kind == "" {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d].configRef.kind", fnType, idx),
			Reason: "configRef must specify `kind`",
		}
	}
	if r.Name == "" {
		return &ValidateError{
			Field:  fmt.Sprintf("pipeline.%s[%d].configRef.name", fnType, idx),
			Reason: "configRef must specify `name`",
		}
	}
	return nil
}

// GetValidatedFnConfigFromPath validates the functionConfig at the path specified by
// the package path (pkgPath) and configPath, returning the functionConfig as an
// RNode if the validation is successful.
func GetValidatedFnConfigFromPath(fsys filesys.FileSystem, pkgPath UniquePath, configPath string) (*yaml.RNode, error) {
	path := filepath.Join(string(pkgPath), configPath)
	file, err := fsys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open functionConfig from path %q: %w", configPath, err)
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
		return nil, fmt.Errorf("functionConfig %q: %w", configPath, err)
	}
	return nodes[0], nil
}

// AreKRM validates if given resources are valid KRM resources.
func AreKRM(nodes []*yaml.RNode) error {
	for i := range nodes {
		if err := IsKRM(nodes[i]); err != nil {
			path, _, _ := kioutil.GetFileAnnotations(nodes[i])
			return fmt.Errorf("%s: %w", path, err)
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
		// https://github.com/kptdev/kpt/issues/2388
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

		if slices.Contains(RecognizedKustomizationFileNames(), resourceFile) {
			return true
		}
	}
	meta, err := n.GetMeta()
	if err != nil {
		return false
	}

	if strings.HasPrefix(meta.APIVersion, kustomizationAPIGroup) {
		return true
	}

	if meta.APIVersion == "" && meta.Kind == kustomizationKind {
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
	fmt.Fprintf(&sb, "Kptfile is invalid:\nField: `%s`\n", e.Field)
	if e.Value != "" {
		fmt.Fprintf(&sb, "Value: %q\n", e.Value)
	}
	fmt.Fprintf(&sb, "Reason: %s\n", e.Reason)
	return sb.String()
}

// RecognizedKustomizationFileNames taken from sigs.k8s.io/kustomize/api@v0.21.1/konfig/general.go
// to avoid dependency.
func RecognizedKustomizationFileNames() []string {
	return []string{
		"kustomization.yaml",
		"kustomization.yml",
		"Kustomization",
	}
}
