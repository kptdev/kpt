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

func SetReplicas(pkgPath string, cmd *cobra.Command) error {
	h := helper{
		Id:      "set-replicas",
		pkgPath: pkgPath,
		enabled: ReplicasField,
	}
	if !IsWildcardPath(pkgPath) {
		kptfile, err := kptfileutil.ReadFile(pkgPath)
		if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
			return nil
		}
	}

	if enabled, err := h.isEnabled(); err != nil || !enabled {
		return err
	}

	c := &cobra.Command{
		Use:   "replicas NAME",
		Short: "Set the replicas for a Resource",
		Long: fmt.Sprintf(`Set the container image on a workload

Args:

  NAME:
    Name of the Resource and Container on which to set the image.

Command is enabled for a package by having a Resource with the field: %s
`, strings.Join(ContainerField, ".")),
		Example: fmt.Sprintf(`kpt %s set replicas NAME --value VALUE`, pkgPath),
		Args:    cobra.ExactArgs(1),
	}
	h.command = c

	value := c.Flags().Int("value", 0, "the new replicas value")
	_ = c.MarkFlagRequired("value")

	c.RunE = func(cmd *cobra.Command, args []string) error {
		h.name = args[0]
		h.field = ReplicasField
		h.setVal = fmt.Sprintf("%d", *value)
		return h.set()
	}
	if pkgPath != "" {
		AddCommand(cmd, c, []string{pkgPath, "set"})
	} else {
		AddCommand(cmd, c, []string{"set"})
	}
	return nil
}

func GetReplicas(pkgPath string, cmd *cobra.Command) error {
	h := helper{
		Id:      "get-replicas",
		pkgPath: pkgPath,
		enabled: ReplicasField,
	}
	if !IsWildcardPath(pkgPath) {
		kptfile, err := kptfileutil.ReadFile(pkgPath)
		if err == nil && !kptfile.IsDuckCommandEnabled(h.Id) {
			return nil
		}
	}

	if enabled, err := h.isEnabled(); err != nil || !enabled {
		return err
	}

	c := &cobra.Command{
		Use:   "replicas NAME",
		Short: "Get the replicas for a Resource",
		Long: fmt.Sprintf(`Get the container image for a workload

Args:

  NAME:
    Name of the Resource and Container from which to get the image.

Command is enabled for a package by having a Resource with the field: %s
`, strings.Join(ContainerField, ".")),
		Example: fmt.Sprintf(`kpt %s get replicas NAME`, pkgPath),
		Args:    cobra.ExactArgs(1),
	}
	h.command = c

	c.RunE = func(cmd *cobra.Command, args []string) error {
		h.name = args[0]
		h.field = ReplicasField
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
