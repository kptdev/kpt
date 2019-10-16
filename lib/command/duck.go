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

// Package duck contains instances of duck-type commands.
package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/yaml"
)

// DuckCommand is a function which returns a command for the given package if
// it is enabled for the package.  Otherwise it returns an error.
type DuckCommand func(string, *cobra.Command) error

// Commands is the list of duck-typed command functions.
var commands = []DuckCommand{
	GetCpuLimits, GetCpuReservations, GetImage, GetReplicas, GetEnv,
	SetCpuLimits, SetCpuReservations, SetImage, SetReplicas, SetEnv,

	GetMemoryLimits, GetMemoryReservations,
	SetMemoryLimits, SetMemoryReservations,
}

func AddCommands(pkgPath string, root *cobra.Command) error {
	for i := range commands {
		if err := commands[i](pkgPath, root); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
	return nil
}

// ContainerField is the path to the containers field
var ContainerField = []string{"spec", "template", "spec", "containers"}

// ReplicasField is the path to the replicas field
var ReplicasField = []string{"spec", "replicas"}

// ImageField returns the path to the image field for the named container
func ImageField(name string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", name), "image"}
}

// CpuLimitsField returns the path to the cpu-limits field for the named container
func CpuLimitsField(name string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", name), "resources", "limits", "cpu"}
}

// CpuReservationsField returns the path to the cpu-reservations field for the named container
func CpuReservationsField(name string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", name), "resources", "requests", "cpu"}
}

// MemoryLimitsField returns the path to the memory-limits field for the named container
func MemoryLimitsField(name string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", name), "resources", "limits", "memory"}
}

// MemoryReservationsField returns the path to the memory-reservations field for the named
// container
func MemoryReservationsField(name string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", name), "resources", "requests", "memory"}
}

// EnvVarField returns the path to the env field for the named container and env
func EnvVarField(container, variable string) []string {
	return []string{"spec", "template", "spec", "containers",
		fmt.Sprintf("[name=%s]", container), "env",
		fmt.Sprintf("[name=%s]", variable), "value"}
}

// helper simplifies creating duck-typed commands for setting and getting fields
type helper struct {
	command *cobra.Command

	Id string

	// pkgPath is the path to a kpt package
	pkgPath string

	// name is the name of a resource in a kpt package to either set or get fields from
	name string

	// enabled is the path to a field which enables the duck-typed command
	enabled []string

	// field is the path to a field which will be set or gotten
	field []string

	// setVal is the value to set on the field
	setVal string
}

// isEnabled returns nil if the command is enabled
func (h helper) isEnabled() (bool, error) {
	if IsWildcardPath(h.pkgPath) {
		return true, nil
	}

	enabled := false
	err := kio.Pipeline{
		Inputs: []kio.Reader{kio.LocalPackageReader{PackagePath: h.pkgPath}},
		Filters: []kio.Filter{
			filters.MatchFilter{Filters: []yaml.YFilter{{Filter: yaml.Lookup(h.enabled...)}}},
			filters.MatchFilter{Filters: []yaml.YFilter{{Filter: h}}},
		},
		Outputs: []kio.Writer{kio.WriterFunc(func(r []*yaml.RNode) error {
			if len(r) > 0 {
				enabled = true
			}
			return nil
		})},
	}.Execute()
	return enabled, err
}

// GrepFilter returns nil if the command is not enabled for the Resource
func (h helper) Filter(object *yaml.RNode) (*yaml.RNode, error) {
	meta, err := object.GetMeta()
	if err != nil {
		return nil, err
	}
	if _, found := meta.Annotations["kpt.dev/duck/"+h.Id]; found {
		return nil, nil
	}
	return object, nil
}

// get prints the value of the field to stdOut
func (h helper) get() error {
	var inputs []kio.Reader
	var outputs []kio.Writer
	if h.pkgPath != duck {
		// read from package
		rw := &kio.LocalPackageReadWriter{
			NoDeleteFiles: true,
			PackagePath:   h.pkgPath}
		inputs = append(inputs, rw)
		outputs = append(outputs, rw)
	} else {
		// read from stdin
		rw := &kio.ByteReadWriter{
			OmitReaderAnnotations: true,
			KeepReaderAnnotations: true,
			Reader:                h.command.InOrStdin(),
			Writer:                h.command.OutOrStdout(),
		}
		inputs = append(inputs, rw)
		outputs = append(outputs, rw)
	}

	var match []yaml.YFilters
	match = append(match, []yaml.YFilter{
		{Filter: yaml.Lookup("metadata", "name")},
		{Filter: yaml.Match(h.name)},
	})
	match = append(match, []yaml.YFilter{{Filter: yaml.Lookup(h.enabled...)}})
	match = append(match, []yaml.YFilter{{Filter: h}})

	found := false

	err := kio.Pipeline{
		Inputs: inputs,
		Filters: []kio.Filter{
			filters.MatchModifyFilter{
				MatchFilters: match,
				ModifyFilters: []yaml.YFilter{
					{Filter: yaml.Lookup(h.field...)},
					{Filter: yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
						value, err := object.String()
						if err != nil {
							return nil, err
						}
						found = true
						fmt.Fprintf(h.command.OutOrStdout(), "%s\n", strings.TrimSpace(value))
						return nil, nil
					})},
				}},
			filters.FormatFilter{},
		},
		Outputs: outputs,
	}.Execute()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no matching resources")
	}
	return nil
}

// set sets the value of the field to setVal
func (h helper) set() error {
	var inputs []kio.Reader
	var outputs []kio.Writer
	if h.pkgPath != duck {
		// read from package
		rw := &kio.LocalPackageReadWriter{
			NoDeleteFiles: true,
			PackagePath:   h.pkgPath}
		inputs = append(inputs, rw)
		outputs = append(outputs, rw)
	} else {
		// read from stdin
		rw := &kio.ByteReadWriter{
			OmitReaderAnnotations: true,
			KeepReaderAnnotations: true,
			Reader:                h.command.InOrStdin(),
			Writer:                h.command.OutOrStdout(),
		}
		inputs = append(inputs, rw)
		outputs = append(outputs, rw)
	}

	var match []yaml.YFilters
	match = append(match, []yaml.YFilter{
		{Filter: yaml.Lookup("metadata", "name")},
		{Filter: yaml.Match(h.name)},
	})
	match = append(match, []yaml.YFilter{{Filter: yaml.Lookup(h.enabled...)}})
	match = append(match, []yaml.YFilter{{Filter: h}})

	found := false
	foundFunc := func(object *yaml.RNode) (*yaml.RNode, error) {
		found = true
		return object, nil
	}
	match = append(match, []yaml.YFilter{{Filter: yaml.FilterFunc(foundFunc)}})

	err := kio.Pipeline{
		Inputs: inputs,
		Filters: []kio.Filter{
			filters.MatchModifyFilter{
				MatchFilters: match,
				ModifyFilters: []yaml.YFilter{
					{Filter: yaml.LookupCreate(yaml.ScalarNode, h.field...)},
					{Filter: yaml.Set(yaml.NewScalarRNode(h.setVal))},
				}},
			filters.FormatFilter{}},
		Outputs: outputs,
	}.Execute()
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no matching resources")
	}
	return nil
}

var HelpCommand = &cobra.Command{
	Use:   "duck-typed",
	Short: "Duck-typed commands are enabled for packages based off the package's content",
	Long: `Duck-typed commands are enabled for packages based off the package's content.

To see the list of duck-typed and custom commands for a package, provide the package as the
first argument to kpt.

	kpt pkg/ -h

Each package may contain Resources which have commands specific to that Resource -- such
as for getting and setting fields.

Duck-typed commands are enabled for packages by inspecting the Resources in the package,
and identifying which commands may be applied to those Resources.

Commands may be enabled by the presence of specific fields in Resources -- e.g. 'set replicas'
or by the presence of specific Resources types in the package.
`,
	Example: `	# list the commands for a package
	kpt PKG_NAME/ -h
	
	# get help for a specific package subcommand
	kpt PKG_NAME/ set image -h
`,
}

const (
	docs = ""
	duck = "duck"
)

func IsWildcardPath(pkgPath string) bool {
	return pkgPath == docs || pkgPath == duck
}
