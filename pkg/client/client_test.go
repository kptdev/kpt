// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUpdateAnnotation(t *testing.T) {
	testcases := []struct {
		name         string
		annotations  map[string]string
		oldID        string
		newID        string
		shouldUpdate bool
	}{
		{
			name:         "empty owning-inventory is changed to newID",
			annotations:  nil,
			oldID:        "old",
			newID:        "new",
			shouldUpdate: true,
		},
		{
			name: "oldID is changed to newID",
			annotations: map[string]string{
				"config.k8s.io/owning-inventory": "old",
			},
			oldID:        "old",
			newID:        "new",
			shouldUpdate: true,
		},
		{
			name: "non empty unmatched id won't be changed to newID",
			annotations: map[string]string{
				"config.k8s.io/owning-inventory": "random",
			},
			oldID:        "old",
			newID:        "new",
			shouldUpdate: false,
		},
	}
	deployment := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      "deployment",
				"namespace": "test",
			},
		},
	}
	for _, tc := range testcases {
		deployment.SetAnnotations(tc.annotations)
		updated, err := UpdateAnnotation(deployment, tc.oldID, tc.newID)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		if tc.shouldUpdate {
			if !updated {
				t.Errorf("owning-inventory should be updated")
			}
			if deployment.GetAnnotations()["config.k8s.io/owning-inventory"] != tc.newID {
				t.Errorf("the owning-inventory annotation is not correctly updated")
			}
		} else if updated {
			t.Errorf("owning-inventory shouldn't be changed")
		}
	}
}
