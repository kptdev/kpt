// Copyright 2020 Google LLC.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
)

func TestReplaceOwningInventoryID(t *testing.T) {
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
		updated, err := ReplaceOwningInventoryID(deployment, tc.oldID, tc.newID)
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

func TestUpdateLabelsAndAnnotations(t *testing.T) {
	testcases := []struct {
		name        string
		labels      map[string]string
		annotations map[string]string
	}{
		{
			name: "add new annotations",
			annotations: map[string]string{
				"config.k8s.io/owning-inventory": "new",
				"random-key":                     "value",
			},
		},
		{
			name: "remove existing annotations",
			annotations: map[string]string{
				"random-key": "value",
			},
		},
		{
			name: "add new labels",
			labels: map[string]string{
				"old-key":    "old-value",
				"random-key": "value",
			},
		},
		{
			name: "remove existing labels",
			labels: map[string]string{
				"random-key": "value",
			},
		},
	}

	for _, tc := range testcases {
		u := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]interface{}{
					"name":      "deployment",
					"namespace": "test",
					"labels": map[string]interface{}{
						"old-key": "old-value",
					},
					"annotations": map[string]interface{}{
						"config.k8s.io/owning-inventory": "old",
					},
				},
			},
		}
		uCloned := u.DeepCopy()
		if tc.annotations != nil {
			uCloned.SetAnnotations(tc.annotations)
		}
		if tc.labels != nil {
			uCloned.SetLabels(tc.labels)
		}
		err := util.CreateOrUpdateAnnotation(true, uCloned, scheme.DefaultJSONEncoder())
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		err = util.CreateOrUpdateAnnotation(true, u, scheme.DefaultJSONEncoder())
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		if tc.labels != nil {
			err = UpdateLabelsAndAnnotations(u, tc.labels, u.GetAnnotations())
		} else if tc.annotations != nil {
			err = UpdateLabelsAndAnnotations(u, u.GetLabels(), tc.annotations)
		}
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		if !reflect.DeepEqual(u, uCloned) {
			t.Errorf("%s failed: expected %v, but got %v", tc.name, uCloned, u)
		}
	}
}
