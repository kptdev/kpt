// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package flagutils

import (
	"fmt"

	"sigs.k8s.io/cli-utils/pkg/inventory"
)

const (
	InventoryPolicyFlag   = "inventory-policy"
	InventoryPolicyStrict = "strict"
	InventoryPolicyAdopt  = "adopt"
)

func ConvertInventoryPolicy(policy string) (inventory.InventoryPolicy, error) {
	switch policy {
	case InventoryPolicyStrict:
		return inventory.InventoryPolicyMustMatch, nil
	case InventoryPolicyAdopt:
		return inventory.AdoptIfNoInventory, nil
	default:
		return inventory.InventoryPolicyMustMatch, fmt.Errorf(
			"inventory policy must be one of strict, adopt")
	}
}

// PathFromArgs returns the path which is a positional arg from args list
// returns "-" if there is length of args is 0, which implies no path is provided
func PathFromArgs(args []string) string {
	if len(args) == 0 {
		return "-"
	}
	return args[0]
}
