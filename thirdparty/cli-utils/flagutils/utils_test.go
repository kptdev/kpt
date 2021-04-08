// Copyright 2021 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package flagutils

import (
	"fmt"
	"testing"

	"sigs.k8s.io/cli-utils/pkg/inventory"
)

func TestConvertInventoryPolicy(t *testing.T) {
	testcases := []struct {
		value  string
		policy inventory.InventoryPolicy
		err    error
	}{
		{
			value:  "strict",
			policy: inventory.InventoryPolicyMustMatch,
		},
		{
			value:  "adopt",
			policy: inventory.AdoptIfNoInventory,
		},
		{
			value: "random",
			err:   fmt.Errorf("inventory policy must be one of strict, adopt"),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.value, func(t *testing.T) {
			policy, err := ConvertInventoryPolicy(tc.value)
			if tc.err == nil {
				if err != nil {
					t.Errorf("unexpected error %v", err)
				}
				if policy != tc.policy {
					t.Errorf("expected %v but got %v", policy, tc.policy)
				}
			}
			if err == nil && tc.err != nil {
				t.Errorf("expected an error, but not happened")
			}
		})
	}
}
