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

package kptfile

import (
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

type CommandList struct {
	ObjectMetadata   `yaml:"metadata"`
	PipelineCommands []PipelineCommand `yaml:"pipelineCommands"`
	DuckCommands     []DuckCommand     `yaml:"duckCommands"`
}

type ObjectMetadata struct {
	Name string `yaml:"name"`
}

// PipelineCommand defines a command that is dynamically defined as an annotation on a CRD
type PipelineCommand struct {
	// Command is the Command description
	Command Command `yaml:"command"`

	// Pipeline is what is run by the Command
	Pipeline Pipeline `yaml:"pipeline"`
}

// Pipeline contains a list of Transformers that may be applied to a collection of Resources.
type Pipeline struct {

	// Transformers are transformations applied to the Resource Configuration.
	// They are applied in the order they are specified.
	Filters []filters.KFilter `yaml:"filters"`
}

func (p Pipeline) KioFilters() []kio.Filter {
	var f []kio.Filter
	for i := range p.Filters {
		f = append(f, p.Filters[i].Filter)
	}
	return f
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
	// StringSlice defines a string slice flag
	StringSlice = "slice"
)

// InputParameter defines an input parameter that should be registered with the templates.
type InputParameter struct {
	Type InputType `yaml:"type"`

	Name string `yaml:"name"`

	Description string `yaml:"description"`

	// +optional
	Required *bool `yaml:"required"`

	// +optional
	StringValue string `yaml:"stringValue"`

	// +optional
	StringSliceValue []string `yaml:"stringSliceValue"`

	// +optional
	BoolValue bool `yaml:"boolValue"`

	// +optional
	IntValue int32 `yaml:"intValue"`

	// +optional
	FloatValue float64 `yaml:"floatValue"`
}

// Command defines a Command published on a CRD and created as a cobra Command in the cli
type Command struct {
	// Use is the one-line usage message.
	Use string `yaml:"use"`

	// Path is the path to the sub-command.  Omit if the command is directly under the root command.
	// +optional
	Path []string `yaml:"path"`

	// Inputs are the inputs to the pipeline.
	//
	// Example:
	// 		  - name: namespace
	//    		type: String
	//    		stringValue: "default"
	//    		description: "deployment namespace"
	//
	// +optional
	Inputs []InputParameter `yaml:"inputs"`

	// Short is the short description shown in the 'help' output.
	// +optional
	Short string `yaml:"short"`

	// Long is the long message shown in the 'help <this-command>' output.
	// +optional
	Long string `yaml:"long"`

	// Example is examples of how to use the command.
	// +optional
	Example string `yaml:"example"`

	// Deprecated defines, if this command is deprecated and should print this string when used.
	// +optional
	Deprecated string `yaml:"deprecated"`

	// SuggestFor is an array of command names for which this command will be suggested -
	// similar to aliases but only suggests.
	SuggestFor []string `yaml:"suggestFor"`

	// Aliases is an array of aliases that can be used instead of the first word in Use.
	Aliases []string `yaml:"aliases"`

	// Version defines the version for this command. If this value is non-empty and the command does not
	// define a "version" flag, a "version" boolean flag will be added to the command and, if specified,
	// will print content of the "Version" variable.
	// +optional
	Version string `yaml:"version"`

	ExactArgs int `yaml:"exactArgs"`
}

type DuckCommand struct {
	GetCommand Command `yaml:"getCommand"`
	SetCommand Command `yaml:"setCommand"`
	Duck       Duck    `yaml:"duck"`
}

type Duck struct {
	ResourceName string `yaml:"resourceName"`

	EnabledBy []string `yaml:"enabledBy"`

	GetSetField []string `yaml:"getSetField"`

	SetValue string `yaml:"setValue"`
}
