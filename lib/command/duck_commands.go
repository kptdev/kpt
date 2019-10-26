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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kptfile"
	"lib.kpt.dev/yaml"
)

var resourceKinds = []string{"cpu", "memory"}
var resourceTypes = []string{"limits", "requests"}

func AddCommands(pkgPath string, command *cobra.Command) error {
	true := true
	for _, resourceKind := range resourceKinds {
		for _, resourceType := range resourceTypes {
			name := resourceKind + "-" + resourceType
			use := fmt.Sprintf("%s NAME", name)
			dc := DuckCommand{
				DuckCommand: kptfile.DuckCommand{
					GetCommand: kptfile.Command{Use: use, Path: []string{"get"}, ExactArgs: 1},
					SetCommand: kptfile.Command{Use: name, Path: []string{"set"}, ExactArgs: 1,
						Inputs: []kptfile.InputParameter{
							{
								Type:     "string",
								Name:     "value",
								Required: &true,
							},
						},
					},
					Duck: kptfile.Duck{
						ResourceName: "{{ arg 0 }}",
						EnabledBy:    getContainerField(),
						GetSetField: getContainerField(
							"[name={{ arg 0 }}]", "resources", resourceType, resourceKind),
						SetValue: "{{ input \"value\" }}",
					},
				},
				PkgPath: pkgPath,
			}
			if err := dc.RegisterGetSet(command); err != nil {
				return err
			}
		}
	}

	replicas := DuckCommand{
		DuckCommand: kptfile.DuckCommand{
			GetCommand: kptfile.Command{Use: "replicas NAME", Path: []string{"get"}, ExactArgs: 1},
			SetCommand: kptfile.Command{Use: "replicas NAME", Path: []string{"set"}, ExactArgs: 1,
				Inputs: []kptfile.InputParameter{
					{
						Type:     "string",
						Name:     "value",
						Required: &true,
					},
				},
			},
			Duck: kptfile.Duck{
				ResourceName: "{{ arg 0 }}",
				GetSetField:  []string{"spec", "replicas"},
				SetValue:     "{{ input \"value\" }}",
			},
		},
		PkgPath: pkgPath,
	}
	if err := replicas.RegisterGetSet(command); err != nil {
		return err
	}

	env := DuckCommand{
		DuckCommand: kptfile.DuckCommand{
			GetCommand: kptfile.Command{Use: "env NAME", Path: []string{"get"}, ExactArgs: 1,
				Inputs: []kptfile.InputParameter{
					{
						Type:     "string",
						Name:     "name",
						Required: &true,
					},
				},
			},
			SetCommand: kptfile.Command{Use: "env NAME", Path: []string{"set"}, ExactArgs: 1,
				Inputs: []kptfile.InputParameter{
					{
						Type:     "string",
						Name:     "name",
						Required: &true,
					},
					{
						Type:     "string",
						Name:     "value",
						Required: &true,
					},
				},
			},
			Duck: kptfile.Duck{
				ResourceName: "{{ arg 0 }}",
				EnabledBy:    getContainerField(),
				GetSetField: getContainerField(
					"[name={{ arg 0 }}]", "env", "[name={{ input \"name\"}}]", "value"),
				SetValue: "{{ input \"value\" }}",
			},
		},
		PkgPath: pkgPath,
	}
	if err := env.RegisterGetSet(command); err != nil {
		return err
	}

	image := DuckCommand{
		DuckCommand: kptfile.DuckCommand{
			GetCommand: kptfile.Command{Use: "image NAME", Path: []string{"get"}, ExactArgs: 1},
			SetCommand: kptfile.Command{Use: "image NAME", Path: []string{"set"}, ExactArgs: 1,
				Inputs: []kptfile.InputParameter{
					{
						Type:     "string",
						Name:     "value",
						Required: &true,
					},
				},
			},
			Duck: kptfile.Duck{
				ResourceName: "{{ arg 0 }}",
				EnabledBy:    getContainerField(),
				GetSetField:  getContainerField("[name={{ arg 0 }}]", "image"),
				SetValue:     "{{ input \"value\" }}",
			},
		},
		PkgPath: pkgPath,
	}
	if err := image.RegisterGetSet(command); err != nil {
		return err
	}

	// register duck-type commands from the Kptfile
	cmds := getCommands(pkgPath)
	if cmds == nil {
		return nil
	}
	for i := range cmds.DuckCommands {
		duck := DuckCommand{
			DuckCommand: cmds.DuckCommands[i],
			PkgPath:     pkgPath,
		}
		if err := duck.RegisterGetSet(command); err != nil {
			return err
		}
	}

	return nil
}

func getContainerField(subpath ...string) []string {
	return append([]string{"spec", "template", "spec", "containers"}, subpath...)
}

func getCommands(pkgPath string) *kptfile.CommandList {
	f, err := os.Open(filepath.Join(pkgPath, kptfile.KptFileName))

	// if we are in a package subdirectory, find the parent dir with the Kptfile.
	// this is necessary to parse the duck-commands for sub-directories of a package
	for os.IsNotExist(err) && filepath.Dir(pkgPath) != pkgPath {
		pkgPath = filepath.Dir(pkgPath)
		f, err = os.Open(filepath.Join(pkgPath, kptfile.KptFileName))
	}
	if err != nil {
		return nil
	}
	defer f.Close()

	b, err := ioutil.ReadFile(filepath.Join(pkgPath, kptfile.KptFileName))
	if err != nil {
		return nil
	}

	cmds := &kptfile.CommandList{}
	d := yaml.NewDecoder(bytes.NewBuffer(b))
	if err := d.Decode(cmds); err != nil {
		return nil
	}
	return cmds
}
