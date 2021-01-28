// Copyright 2021 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package preprocess

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/provider"
)

func PreProcess(p provider.Provider, inv inventory.InventoryInfo, strategy common.DryRunStrategy) (inventory.InventoryPolicy, error) {
	invClient, err := p.InventoryClient()
	if err != nil {
		return inventory.InventoryPolicyMustMatch, err
	}
	obj, err := invClient.GetClusterInventoryInfo(inv)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return inventory.InventoryPolicyMustMatch, nil
		}
		return inventory.InventoryPolicyMustMatch, err
	}

	if obj == nil {
		return inventory.InventoryPolicyMustMatch, nil
	}

	managedByKey := "apps.kubernetes.io/managed-by"
	managedByVal := "kpt"
	labels := obj.GetLabels()
	val, found := labels[managedByKey]
	if found {
		if val != managedByVal {
			return inventory.InventoryPolicyMustMatch, fmt.Errorf("can't apply the current package since it is managed by %s", val)
		}
		return inventory.InventoryPolicyMustMatch, nil
	}
	labels[managedByKey] = managedByVal
	if strategy.ClientOrServerDryRun() {
		return inventory.AdoptIfNoInventory, nil
	}
	err = invClient.UpdateLabels(inv, labels)
	return inventory.AdoptIfNoInventory, err
}
