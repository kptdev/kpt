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

package duck

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"kpt.dev/util/pkgfile"
	"lib.kpt.dev/custom"
)

func SetEnv(pkgPath string, cmd *cobra.Command) error {
	h := helper{
		Id:      "set-env",
		pkgPath: pkgPath,
		enabled: ContainerField,
	}
	if pkgPath != "" {
		kptfile, err := pkgfile.ReadFile(pkgPath)
		if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
			return nil
		}
	}

	if enabled, err := h.isEnabled(); err != nil || !enabled {
		return err
	}

	c := &cobra.Command{
		Use:   "env NAME",
		Short: "Set an environment variable on a container",
		Long: fmt.Sprintf(`Set an environment variable on a container.

Args:

  NAME:
    Name of the Resource and Container on which to set the environment variable.

Command is enabled for a package by having a Resource with the field: %s
`, strings.Join(ContainerField, ".")),
		Example: fmt.Sprintf(`kpt %s set env NAME --name ENV_NAME --value ENV_VALUE`, pkgPath),
		Args:    cobra.ExactArgs(1),
	}

	name := c.Flags().String("name", "", "the environment variable name")
	_ = c.MarkFlagRequired("name")

	value := c.Flags().String("value", "", "the environment variable value")
	_ = c.MarkFlagRequired("value")

	c.RunE = func(cmd *cobra.Command, args []string) error {
		h.name = args[0]
		h.field = EnvVarField(args[0], *name)
		h.setVal = *value
		return h.set()
	}
	if pkgPath != "" {
		custom.AddCommand(cmd, c, []string{pkgPath, "set"})
	} else {
		custom.AddCommand(cmd, c, []string{"set"})
	}

	return nil
}

func GetEnv(pkgPath string, cmd *cobra.Command) error {
	h := helper{
		Id:      "get-env",
		pkgPath: pkgPath,
		enabled: ContainerField,
	}
	if pkgPath != "" {
		kptfile, err := pkgfile.ReadFile(pkgPath)
		if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
			return nil
		}
	}

	if enabled, err := h.isEnabled(); err != nil || !enabled {
		return err
	}

	c := &cobra.Command{
		Use:   "env NAME",
		Short: "Get an environment variable from a container",
		Long: fmt.Sprintf(`Get an environment variable from a container.

Args:

  NAME:
    Name of the Resource and Container from which to get the environment variable.

Command is enabled for a package by having a Resource with the field: %s
`, strings.Join(ContainerField, ".")),
		Example: fmt.Sprintf(`kpt %s get env NAME --name ENV_NAME`, pkgPath),
		Args:    cobra.ExactArgs(1),
	}

	name := c.Flags().String("name", "", "the environment variable name")
	_ = c.MarkFlagRequired("name")

	c.RunE = func(cmd *cobra.Command, args []string) error {
		h.name = args[0]
		h.field = EnvVarField(args[0], *name)
		h.command = cmd
		return h.get()
	}
	if pkgPath != "" {
		custom.AddCommand(cmd, c, []string{pkgPath, "get"})
	} else {
		custom.AddCommand(cmd, c, []string{"get"})
	}
	return nil
}
