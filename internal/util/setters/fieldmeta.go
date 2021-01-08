package setters

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/go-openapi/spec"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FieldMeta contains metadata that may be attached to fields as comments
type FieldMeta struct {
	Schema spec.Schema

	Extensions XKustomize

	SettersSchema *spec.Schema
}

type XKustomize struct {
	SetBy               string               `yaml:"setBy,omitempty" json:"setBy,omitempty"`
	PartialFieldSetters []PartialFieldSetter `yaml:"partialSetters,omitempty" json:"partialSetters,omitempty"`
	FieldSetter         *PartialFieldSetter  `yaml:"setter,omitempty" json:"setter,omitempty"`
}

// PartialFieldSetter defines how to set part of a field rather than the full field
// value.  e.g. the tag part of an image field
type PartialFieldSetter struct {
	// Name is the name of this setter.
	Name string `yaml:"name" json:"name"`

	// Value is the current value that has been set.
	Value string `yaml:"value" json:"value"`
}

// IsEmpty returns true if the FieldMeta has any empty Schema
func (fm *FieldMeta) IsEmpty() bool {
	if fm == nil {
		return true
	}
	return reflect.DeepEqual(fm.Schema, spec.Schema{})
}

// Read reads the FieldMeta from a node
func (fm *FieldMeta) Read(n *yaml.RNode) error {
	// check for metadata on head and line comments
	comments := []string{n.YNode().LineComment, n.YNode().HeadComment}
	for _, c := range comments {
		if c == "" {
			continue
		}
		c := strings.TrimLeft(c, "#")
		if !fm.processShortHand(c) {
			continue
		}
		fe := fm.Schema.VendorExtensible.Extensions["x-kustomize"]
		if fe == nil {
			return nil
		}
		b, err := json.Marshal(fe)
		if err != nil {
			return errors.Wrap(err)
		}
		return json.Unmarshal(b, &fm.Extensions)
	}
	return nil
}

// processShortHand parses the comment for short hand ref, loads schema to fm
// and returns true if successful, returns false for any other cases and not throw
// error, as the comment might not be a setter ref
func (fm *FieldMeta) processShortHand(comment string) bool {
	input := map[string]string{}
	err := json.Unmarshal([]byte(comment), &input)
	if err != nil {
		return false
	}
	name := input[setterRef]
	if name == "" {
		return false
	}

	// check if setter with the name exists, else check for a substitution
	// setter and substitution can't have same name in shorthand
	name = strings.TrimSuffix(strings.TrimPrefix(name, "${"), "}")
	setterRef, err := spec.NewRef(DefinitionsPrefix + SetterDefinitionPrefix + name)
	if err != nil {
		return false
	}

	setterRefBytes, err := setterRef.MarshalJSON()
	if err != nil {
		return false
	}

	if _, err := openapi.Resolve(&setterRef, fm.SettersSchema); err == nil {
		setterErr := fm.Schema.UnmarshalJSON(setterRefBytes)
		return setterErr == nil
	}

	substRef, err := spec.NewRef(DefinitionsPrefix + SubstitutionDefinitionPrefix + name)
	if err != nil {
		return false
	}

	substRefBytes, err := substRef.MarshalJSON()
	if err != nil {
		return false
	}

	if _, err := openapi.Resolve(&substRef, fm.SettersSchema); err == nil {
		substErr := fm.Schema.UnmarshalJSON(substRefBytes)
		return substErr == nil
	}
	return false
}

// FieldValueType defines the type of input to register
type FieldValueType string

const (
	// CLIDefinitionsPrefix is the prefix for cli definition keys.
	CLIDefinitionsPrefix = "io.k8s.cli."

	// SetterDefinitionPrefix is the prefix for setter definition keys.
	SetterDefinitionPrefix = CLIDefinitionsPrefix + "setters."

	// SubstitutionDefinitionPrefix is the prefix for substitution definition keys.
	SubstitutionDefinitionPrefix = CLIDefinitionsPrefix + "substitutions."

	// DefinitionsPrefix is the prefix used to reference definitions in the OpenAPI
	DefinitionsPrefix = "#/definitions/"
)

// setterRef is the reference to setter pattern
var setterRef = "$kpt-set"
