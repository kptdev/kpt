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

package custom

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kio"
	"lib.kpt.dev/yaml"
)

// CommandBuilder builds cobra.Commands from ResourceCommand declarations
type CommandBuilder struct {
	PkgPath string
	CmdPath []string
	RootCmd *cobra.Command
	Name    string
}

func (c CommandBuilder) BuildCommands() error {
	b, err := ioutil.ReadFile(filepath.Join(c.PkgPath, "Kptfile"))
	if err != nil {
		return err
	}

	cmds := &CommandList{}
	d := yaml.NewDecoder(bytes.NewBuffer(b))
	if err := d.Decode(cmds); err != nil {
		return err
	}
	c.Name = cmds.Name
	for i := range cmds.Commands {
		rc := cmds.Commands[i]
		if err := c.BuildCommand(rc); err != nil {
			return err
		}
	}
	return nil
}

// Build builds the new command and adds it to root under its path
func (c CommandBuilder) BuildCommand(rc ResourceCommand) error {
	cbra, inputs, err := parse(rc.Command)
	if err != nil {
		return err
	}
	b := &bytes.Buffer{}
	e := yaml.NewEncoder(b)
	if err := e.Encode(rc.Pipeline); err != nil {
		return err
	}
	if err := e.Close(); err != nil {
		return err
	}
	dr := cbra.Flags().Bool(
		"dry-run", false, "if true, don't write the updates")
	pp := cbra.Flags().Bool(
		"show-pipeline", false, "if true, print the pipeline")
	if len(os.Args) > 2 {
		cbra.SetArgs(os.Args[2:])
	} else {
		cbra.SetArgs([]string{"help"})
	}
	cbra.RunE = func(cmd *cobra.Command, args []string) error {
		// fill the arg and input references into the pipeline using go templates
		o := &bytes.Buffer{}
		t, err := template.New(rc.Command.Use).
			Funcs(template.FuncMap{
				"input": inputs.Input,
				"arg":   func(i int) string { return args[i] },
				"pkg":   func() string { return c.Name },
			}).
			Parse(b.String())
		if err != nil {
			return err
		}
		if err := t.Execute(o, inputs); err != nil {
			return err
		}
		if *pp {
			fmt.Println(o.String())
		}

		d := yaml.NewDecoder(o)
		p := &Pipeline{}
		if err := d.Decode(p); err != nil {
			return err
		}

		var outputs []kio.Writer
		if *dr {
			outputs = append(outputs, kio.ByteWriter{Writer: cmd.OutOrStdout()})
		} else {
			outputs = append(outputs, kio.LocalPackageWriter{PackagePath: filepath.Join(c.PkgPath)})
		}
		return kio.Pipeline{
			Inputs:  []kio.Reader{kio.LocalPackageReader{PackagePath: c.PkgPath}},
			Filters: p.kioFilters(),
			Outputs: outputs,
		}.Execute()
	}
	addCommand(c.RootCmd, cbra, append(c.CmdPath, rc.Command.Path...))
	return nil
}

// Inputs contains flag values setup for the cobra command
type Inputs struct {
	// Strings contains a map of flag names to string values
	Strings map[string]*string

	// Ints contains a map of flag names to int values
	Ints map[string]*int32

	// Bools contains a map of flag names to bool values
	Bools map[string]*bool

	// Floats contains a map of flag names to flat values
	Floats map[string]*float64

	// StringSlices contains a map of flag names to string slice values
	StringSlices map[string]*[]string
}

// Input returns the string value for a flag input
func (i Inputs) Input(key string) string {
	if v, found := i.Strings[key]; found {
		return fmt.Sprintf("%v", *v)
	}

	if v, found := i.Ints[key]; found {
		return fmt.Sprintf("%v", *v)
	}

	if v, found := i.Bools[key]; found {
		return fmt.Sprintf("%v", *v)
	}

	if v, found := i.Floats[key]; found {
		return fmt.Sprintf("%v", *v)
	}

	if v, found := i.StringSlices[key]; found {
		return fmt.Sprintf("%v", strings.Join(*v, ","))
	}
	return ""
}

// parse parses cmd into a cobra.Command
func parse(cmd Command) (*cobra.Command, Inputs, error) {
	inputs := Inputs{}

	// create the cobra command by copying values from the cli
	cbra := &cobra.Command{
		Use:        cmd.Use,
		Short:      cmd.Short,
		Long:       cmd.Long,
		Example:    cmd.Example,
		Version:    cmd.Version,
		Deprecated: cmd.Deprecated,
		Aliases:    cmd.Aliases,
		SuggestFor: cmd.SuggestFor,
	}

	// Register the cobra Inputs in the values structure
	for i := range cmd.Inputs {
		cmdFlag := cmd.Inputs[i]
		switch cmdFlag.Type {
		case String:
			if inputs.Strings == nil {
				inputs.Strings = map[string]*string{}
			}
			// Create a string flag and register it
			inputs.Strings[cmdFlag.Name] = cbra.Flags().String(cmdFlag.Name,
				cmdFlag.StringValue, cmdFlag.Description)
		case StringSlice:
			if inputs.StringSlices == nil {
				inputs.StringSlices = map[string]*[]string{}
			}
			// Create a string slice flag and register it
			inputs.StringSlices[cmdFlag.Name] = cbra.Flags().StringSlice(
				cmdFlag.Name, cmdFlag.StringSliceValue, cmdFlag.Description)
		case Int:
			if inputs.Ints == nil {
				inputs.Ints = map[string]*int32{}
			}
			// Create an int flag and register it
			inputs.Ints[cmdFlag.Name] = cbra.Flags().Int32(cmdFlag.Name, cmdFlag.IntValue, cmdFlag.Description)
		case Float:
			if inputs.Floats == nil {
				inputs.Floats = map[string]*float64{}
			}
			// Create a float flag and register it
			inputs.Floats[cmdFlag.Name] = cbra.Flags().Float64(cmdFlag.Name, cmdFlag.FloatValue, cmdFlag.Description)
		case Bool:
			if inputs.Bools == nil {
				inputs.Bools = map[string]*bool{}
			}
			// Create a bool flag and register it
			inputs.Bools[cmdFlag.Name] = cbra.Flags().Bool(cmdFlag.Name, cmdFlag.BoolValue, cmdFlag.Description)
		}
		if cmdFlag.Required != nil && *cmdFlag.Required {
			if err := cbra.MarkFlagRequired(cmdFlag.Name); err != nil {
				return nil, Inputs{}, err
			}
		}
	}
	return cbra, inputs, nil
}

// addCommand adds the subcmd to root at the provided path.
// An empty path will add subcmd as a sub-command of root.
func addCommand(root, subcmd *cobra.Command, path []string) {
	next := root
	// For each element on the Path
	for i := range path {
		p := path[i]
		// Make sure the subcommand exists
		found := false
		for i := range next.Commands() {
			c := next.Commands()[i]
			if c.Use == p {
				// Found, continue on to next part of the Path
				next = c
				found = true
				break
			}
		}

		if !found {
			// Missing, create the sub-command
			cbra := &cobra.Command{Use: p}
			next.AddCommand(cbra)
			next = cbra
		}
	}

	next.AddCommand(subcmd)
}
