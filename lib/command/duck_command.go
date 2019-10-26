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

package command

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/kio/filters"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/kptfile/kptfileutil"
	"lib.kpt.dev/yaml"
)

// DuckCommand simplifies creating duck-typed commands for setting and getting fields
type DuckCommand struct {
	kptfile.DuckCommand `yaml:",inline"`

	PkgPath string
}

func (h DuckCommand) RegisterGetSet(root *cobra.Command) error {
	if len(h.Duck.EnabledBy) == 0 {
		h.Duck.EnabledBy = h.Duck.GetSetField
	}

	if h.GetCommand.Use != "" {
		getCmd, err := h.register("get")
		if err != nil {
			return err
		}
		AddCommand(root, getCmd, append([]string{h.PkgPath}, h.GetCommand.Path...))
	}

	if h.SetCommand.Use != "" {
		setCmd, err := h.register("set")
		if err != nil {
			return err
		}
		AddCommand(root, setCmd, append([]string{h.PkgPath}, h.SetCommand.Path...))
	}

	return nil
}

// Get prints the value of the GetSetField to stdOut
func (h DuckCommand) register(kind string) (*cobra.Command, error) {
	// check if the command is enabled
	if enabled, err := h.isEnabled(kind); err != nil || !enabled {
		return nil, err
	}

	var cmd kptfile.Command
	switch kind {
	case "get":
		cmd = h.GetCommand
	case "set":
		cmd = h.SetCommand
	}

	// parse the DuckCommand into a cobra command, initializing flags
	c, inputs, err := parse(cmd)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}
	// encode the Command as a string template so that the parsed flags and
	// arguments will be substituted into the DuckCommand
	e := yaml.NewEncoder(b)
	if err := e.Encode(h.Duck); err != nil {
		return nil, err
	}
	if err := e.Close(); err != nil {
		return nil, err
	}

	c.RunE = func(cmd *cobra.Command, args []string) error {
		// substitute the flags and arguments using go templating
		o := &bytes.Buffer{}
		t, err := template.New(cmd.Use).
			Funcs(template.FuncMap{
				"input": inputs.Input,
				"arg":   func(i int) string { return args[i] },
			}).
			Parse(b.String())
		if err != nil {
			return err
		}
		if err := t.Execute(o, inputs); err != nil {
			return err
		}

		// write the result back to DuckCommand
		d := yaml.NewDecoder(o)
		if err := d.Decode(&h.Duck); err != nil {
			return err
		}

		return h.execute(c, kind)
	}

	if cmd.ExactArgs != 0 {
		c.Args = cobra.ExactArgs(cmd.ExactArgs)
	}

	return c, nil
}

// IsEnabled returns true if the command is enabled for the package
func (h DuckCommand) isEnabled(kind string) (bool, error) {
	if IsWildcardPath(h.PkgPath) {
		return true, nil
	}

	kf, err := kptfileutil.ReadFile(h.PkgPath)
	if err == nil && !kf.IsDuckCommandEnabled(h.getCommandId(kind)) {
		// command explicitly disabled for this package -- disabled in Kptfile
		return false, nil
	}

	enabled := false
	err = kio.Pipeline{
		Inputs: []kio.Reader{kio.LocalPackageReader{PackagePath: h.PkgPath}},
		Filters: []kio.Filter{
			filters.MatchFilter{Filters: []yaml.YFilter{{Filter: yaml.Lookup(h.Duck.EnabledBy...)}}},
			filters.MatchFilter{Filters: []yaml.YFilter{{Filter: h.matchNotDisabled(kind)}}},
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

func (h DuckCommand) execute(command *cobra.Command, kind string) error {
	var found bool
	var fltrs []yaml.YFilter
	switch kind {
	case "get":
		fltrs = h.getFilters(command, &found)
	case "set":
		fltrs = h.setFilters(command, &found)
	}

	input, output := h.inputOutput(command, kind)
	pipeline := kio.Pipeline{
		Inputs: input,
		Filters: []kio.Filter{
			filters.MatchModifyFilter{
				MatchFilters:  h.matchResourcesFilters(kind), // Find the Resources to update
				ModifyFilters: fltrs,                         // Do the work
			},
			filters.FormatFilter{}, // Format Resources before writing them out
		},
		Outputs: output,
	}
	if err := pipeline.Execute(); err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no matching resources")
	}
	return nil
}

func (h DuckCommand) getFilters(command *cobra.Command, found *bool) []yaml.YFilter {
	printFunc := func(object *yaml.RNode) (*yaml.RNode, error) {
		value, err := object.String()
		if err != nil {
			return nil, err
		}
		*found = true
		fmt.Fprintf(command.OutOrStdout(), "%s\n", strings.TrimSpace(value))
		return nil, nil
	}

	return []yaml.YFilter{
		{Filter: yaml.Lookup(h.Duck.GetSetField...)},
		{Filter: yaml.FilterFunc(printFunc)},
	}
}

func (h DuckCommand) setFilters(command *cobra.Command, found *bool) []yaml.YFilter {
	foundFunc := func(object *yaml.RNode) (*yaml.RNode, error) {
		*found = true
		return object, nil
	}
	return []yaml.YFilter{
		{Filter: yaml.FilterFunc(foundFunc)},
		{Filter: yaml.LookupCreate(yaml.ScalarNode, h.Duck.GetSetField...)},
		{Filter: yaml.Set(yaml.NewScalarRNode(h.Duck.SetValue))},
	}
}

// matchResourcesFilters returns a set of filters for filtering Resources
// that match the DuckCommand
func (h DuckCommand) matchResourcesFilters(kind string) []yaml.YFilters {
	var match []yaml.YFilters

	// match the name
	match = append(match, []yaml.YFilter{
		{Filter: yaml.Lookup("metadata", "name")},
		{Filter: yaml.Match(h.Duck.ResourceName)},
	})

	// match the field that enables the command
	match = append(match, []yaml.YFilter{{Filter: yaml.Lookup(h.Duck.EnabledBy...)}})

	// remove objects that have this duck-command disabled
	match = append(match, []yaml.YFilter{{Filter: h.matchNotDisabled(kind)}})
	return match
}

func (h DuckCommand) getCommandId(kind string) string {
	switch kind {
	case "get":
		return kind + " " + strings.Split(h.GetCommand.Use, " ")[0]
	case "set":
		return kind + " " + strings.Split(h.SetCommand.Use, " ")[0]
	}
	return ""
}

func (h DuckCommand) matchNotDisabled(kind string) yaml.Filter {
	return yaml.FilterFunc(
		func(object *yaml.RNode) (*yaml.RNode, error) {
			meta, err := object.GetMeta()
			if err != nil {
				return nil, err
			}
			id := h.getCommandId(kind)
			if _, found := meta.Annotations["kpt.dev/disable-duck/"+id]; found {
				return nil, nil
			}
			return object, nil
		})
}

// inputOutput returns the readers and writers for a pipeline
func (h DuckCommand) inputOutput(command *cobra.Command, kind string) ([]kio.Reader, []kio.Writer) {
	if h.PkgPath != Duck {
		// read / write files
		rw := &kio.LocalPackageReadWriter{
			NoDeleteFiles: true,
			PackagePath:   h.PkgPath,
		}
		if kind == "get" {
			return []kio.Reader{rw}, []kio.Writer{}
		}
		return []kio.Reader{rw}, []kio.Writer{rw}
	}

	// read / write stdin + stdout
	rw := &kio.ByteReadWriter{
		OmitReaderAnnotations: true,
		KeepReaderAnnotations: true,
		Reader:                command.InOrStdin(),
		Writer:                command.OutOrStdout(),
	}
	if kind == "get" {
		return []kio.Reader{rw}, []kio.Writer{}
	}
	return []kio.Reader{rw}, []kio.Writer{rw}
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
	Docs = ""
	Duck = "duck"
)

func IsWildcardPath(pkgPath string) bool {
	return pkgPath == Docs || pkgPath == Duck
}
