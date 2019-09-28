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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"lib.kpt.dev/kptfile/kptfileutil"
)

func SetResources(resourceName, resourceType string,
	f func(name string) []string) func(string, *cobra.Command) error {
	return func(pkgPath string, cmd *cobra.Command) error {
		h := helper{
			Id:      "set-" + resourceName + "-" + resourceType,
			pkgPath: pkgPath,
			enabled: ContainerField,
		}
		if pkgPath != "" {
			kptfile, err := kptfileutil.ReadFile(pkgPath)
			if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
				return nil
			}
		}

		if enabled, err := h.isEnabled(); err != nil || !enabled {
			return err
		}

		n := fmt.Sprintf("%s-%s", resourceName, resourceType)
		c := &cobra.Command{
			Use:   n + " NAME",
			Short: "Set " + n + " for a container",
			Long: fmt.Sprintf(`Set %s for a container.

Args:

  NAME:
    Name of the Resource and Container on which to set %s.

Command is enabled for a package by having a Resource with the field: %s
`, n, n, strings.Join(ContainerField, ".")),
			Example: fmt.Sprintf(`kpt %s set %s NAME --value VALUE`, pkgPath, n),
			Args:    cobra.ExactArgs(1),
		}

		val := c.Flags().String("value", "", "the new value")
		_ = c.MarkFlagRequired("value")

		c.RunE = func(cmd *cobra.Command, args []string) error {
			h.name = args[0]
			h.field = f(args[0])
			h.setVal = *val
			return h.set()
		}
		if pkgPath != "" {
			AddCommand(cmd, c, []string{pkgPath, "set"})
		} else {
			AddCommand(cmd, c, []string{"set"})
		}

		return nil
	}
}

func GetResources(resourceName, resourceType string,
	f func(name string) []string) func(pkgPath string, cmd *cobra.Command) error {
	return func(pkgPath string, cmd *cobra.Command) error {
		h := helper{
			Id:      "get-" + resourceName + "-" + resourceType,
			pkgPath: pkgPath,
			enabled: ContainerField,
		}
		if pkgPath != "" {
			kptfile, err := kptfileutil.ReadFile(pkgPath)
			if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
				return nil
			}
		}

		if enabled, err := h.isEnabled(); err != nil || !enabled {
			return err
		}

		n := fmt.Sprintf("%s-%s", resourceName, resourceType)
		c := &cobra.Command{
			Use:   n + " NAME",
			Short: "Get " + n + " for a container",
			Long: fmt.Sprintf(`Get %s for a container.

Args:

  NAME:
    Name of the Resource and Container from which to get %s.

Command is enabled for a package by having a Resource with the field: %s
`, n, n, strings.Join(ContainerField, ".")),
			Example: fmt.Sprintf(`kpt %s get %s NAME`, pkgPath, n),
			Args:    cobra.ExactArgs(1),
		}

		c.RunE = func(cmd *cobra.Command, args []string) error {
			h.name = args[0]
			h.field = f(args[0])
			h.command = cmd
			return h.get()
		}
		if pkgPath != "" {
			AddCommand(cmd, c, []string{pkgPath, "get"})
		} else {
			AddCommand(cmd, c, []string{"get"})
		}

		return nil
	}
}

func SetCpuLimits(pkgPath string, cmd *cobra.Command) error {
	return SetResources("cpu", "limits", CpuLimitsField)(pkgPath, cmd)
}

func GetCpuLimits(pkgPath string, cmd *cobra.Command) error {
	return GetResources("cpu", "limits", CpuLimitsField)(pkgPath, cmd)
}

func SetCpuReservations(pkgPath string, cmd *cobra.Command) error {
	return SetResources("cpu", "requests", CpuReservationsField)(pkgPath, cmd)
}

func GetCpuReservations(pkgPath string, cmd *cobra.Command) error {
	return GetResources("cpu", "requests", CpuReservationsField)(pkgPath, cmd)
}

func SetMemoryLimits(pkgPath string, cmd *cobra.Command) error {
	return SetResources("memory", "limits", MemoryLimitsField)(pkgPath, cmd)
}

func GetMemoryLimits(pkgPath string, cmd *cobra.Command) error {
	return GetResources("memory", "limits", MemoryLimitsField)(pkgPath, cmd)
}

func SetMemoryReservations(pkgPath string, cmd *cobra.Command) error {
	return SetResources("memory", "requests", MemoryReservationsField)(pkgPath, cmd)
}

func GetMemoryReservations(pkgPath string, cmd *cobra.Command) error {
	return GetResources("memory", "requests", MemoryReservationsField)(pkgPath, cmd)
}
