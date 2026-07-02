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

// +kubebuilder:object:generate=true
package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	schema "github.com/kptdev/kpt/api/schema/v1"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.21.0 object:headerFile="../../../hack/boilerplate.go.txt",year=$YEAR_GEN

// Result contains the structured result from an individual function
type Result struct {
	// Image is the full name of the image that generates this result
	// Image and Exec are mutually exclusive
	Image string `yaml:"image,omitempty"`
	// ExecPath is the the absolute os-specific path to the executable file
	// If user provides an executable file with commands, ExecPath should
	// contain the entire input string.
	ExecPath string `yaml:"exec,omitempty"`
	// TODO(droot): This is required for making structured results subpackage aware.
	// Enable this once test harness supports filepath based assertions.
	// Pkg is OS specific Absolute path to the package.
	// Pkg string `yaml:"pkg,omitempty"`
	// Stderr is the content in function stderr
	Stderr string `yaml:"stderr,omitempty"`
	// ExitCode is the exit code from running the function
	ExitCode int `yaml:"exitCode"`
	// Results is the list of results for the function
	Results []ResultItem `yaml:"results,omitempty"`
}

const (
	// Deprecated: prefer ResultListGVK
	ResultListKind = "FunctionResultList"
	// Deprecated: prefer ResultListGVK
	ResultListGroup = "kpt.dev"
	// Deprecated: prefer ResultListGVK
	ResultListVersion = "v1"
	// Deprecated: prefer ResultListGVK
	ResultListAPIVersion = ResultListGroup + "/" + ResultListVersion
)

// KptFileGVK is the GroupVersionKind of FunctionResultList objects
func ResultListGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "kpt.dev",
		Version: "v1",
		Kind:    "FunctionResultList",
	}
}

// ResultList contains aggregated results from multiple functions
type ResultList struct {
	yaml.ResourceMeta `yaml:",inline"`
	// ExitCode is the exit code of kpt command
	ExitCode int `yaml:"exitCode"`
	// Items contain a list of function result
	Items []Result `yaml:"items,omitempty"`
}

// NewResultList returns an instance of ResultList with metadata
// field populated.
func NewResultList() *ResultList {
	return &ResultList{
		ResourceMeta: yaml.ResourceMeta{
			TypeMeta: yaml.TypeMeta{
				APIVersion: ResultListAPIVersion,
				Kind:       ResultListKind,
			},
			ObjectMeta: yaml.ObjectMeta{
				NameMeta: yaml.NameMeta{
					Name: "fnresults",
				},
			},
		},
		Items: []Result{},
	}
}

// ResultItem is a modified version of sigs.k8s.io/kustomize/kyaml/fn/framework.Result
// with a simplified Field field.
type ResultItem struct {
	Message string `yaml:"message,omitempty" json:"message,omitempty"`

	Severity framework.Severity `yaml:"severity,omitempty" json:"severity,omitempty"`

	ResourceRef *yaml.ResourceIdentifier `yaml:"resourceRef,omitempty" json:"resourceRef,omitempty"`

	Field *Field `yaml:"field,omitempty" json:"field,omitempty"`

	File *framework.File `yaml:"file,omitempty" json:"file,omitempty"`

	Tags map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// String provides a human-readable message for the result item
func (i *ResultItem) String() string {
	identifier := i.ResourceRef
	var idStringList []string
	if identifier != nil {
		if identifier.APIVersion != "" {
			idStringList = append(idStringList, identifier.APIVersion)
		}
		if identifier.Kind != "" {
			idStringList = append(idStringList, identifier.Kind)
		}
		if identifier.Namespace != "" {
			idStringList = append(idStringList, identifier.Namespace)
		}
		if identifier.Name != "" {
			idStringList = append(idStringList, identifier.Name)
		}
	}
	formatString := "[%s]"
	severity := i.Severity
	// We default Severity to Info when converting a result to a message.
	if i.Severity == "" {
		severity = framework.Info
	}
	list := []any{severity}
	if len(idStringList) > 0 {
		formatString += " %s"
		list = append(list, strings.Join(idStringList, "/"))
	}
	if i.Field != nil {
		formatString += " %s"
		list = append(list, i.Field.Path)
	}
	formatString += ": %s"
	list = append(list, i.Message)
	return fmt.Sprintf(formatString, list...)
}

// Field is a modified version of sigs.k8s.io/kustomize/kyaml/fn/framework.Field
// where CurrentValue and ProposedValue are strings instead of interface{} values.
type Field struct {
	Path string `yaml:"path,omitempty" json:"path,omitempty"`

	CurrentValue string `yaml:"currentValue,omitempty" json:"currentValue,omitempty"`

	ProposedValue string `yaml:"proposedValue,omitempty" json:"proposedValue,omitempty"`
}

var _ json.Unmarshaler = &Field{}
var _ yaml.Unmarshaler = &Field{}

func (in *Field) UnmarshalJSON(data []byte) error {
	rNode, err := yaml.Parse(string(data))
	if err != nil {
		return fmt.Errorf("error parsing `field`: %v", err)
	}

	return in.unmarshalRNode(rNode)
}

func (in *Field) UnmarshalYAML(value *yaml.Node) error {
	rNode := yaml.NewRNode(value)

	return in.unmarshalRNode(rNode)
}

func (in *Field) unmarshalRNode(rNode *yaml.RNode) error {
	if path, err := rNode.GetString("path"); err == nil {
		in.Path = strings.TrimSpace(path)
	}

	if currentValue, err := rNode.Pipe(yaml.Lookup("currentValue")); err == nil && currentValue.YNode() != nil {
		switch currentValue.YNode().Kind {
		case yaml.ScalarNode:
			in.CurrentValue = currentValue.YNode().Value
		default:
			in.CurrentValue, err = currentValue.String()
			if err != nil {
				return fmt.Errorf("error parsing `field.currentValue`: %v", err)
			}
		}

		in.CurrentValue = strings.TrimSpace(in.CurrentValue)
	}

	if proposedValue, err := rNode.Pipe(yaml.Lookup("proposedValue")); err == nil && proposedValue.YNode() != nil {
		switch proposedValue.YNode().Kind {
		case yaml.ScalarNode:
			in.ProposedValue = proposedValue.YNode().Value
		default:
			in.ProposedValue, err = proposedValue.String()
			if err != nil {
				return fmt.Errorf("error parsing `field.proposedValue`: %v", err)
			}
		}

		in.ProposedValue = strings.TrimSpace(in.ProposedValue)
	}

	return nil
}
