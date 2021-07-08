// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package table

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	pe "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestColumnDefs(t *testing.T) {
	testCases := map[string]struct {
		columnName     string
		resource       Resource
		columnWidth    int
		expectedOutput string
	}{
		"namespace": {
			columnName: "namespace",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Identifier: object.ObjMetadata{
						Namespace: "Foo",
					},
				},
			},
			columnWidth:    10,
			expectedOutput: "Foo",
		},
		"namespace trimmed": {
			columnName: "namespace",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Identifier: object.ObjMetadata{
						Namespace: "ICanHearTheHeartBeatingAsOne",
					},
				},
			},
			columnWidth:    10,
			expectedOutput: "ICanHearTh",
		},
		"resource": {
			columnName: "resource",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Identifier: object.ObjMetadata{
						Name: "YoLaTengo",
						GroupKind: schema.GroupKind{
							Kind: "RoleBinding",
						},
					},
				},
			},
			columnWidth:    40,
			expectedOutput: "RoleBinding/YoLaTengo",
		},
		"resource trimmed": {
			columnName: "resource",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Identifier: object.ObjMetadata{
						Name: "SlantedAndEnchanted",
						GroupKind: schema.GroupKind{
							Kind: "Pavement",
						},
					},
				},
			},
			columnWidth:    25,
			expectedOutput: "Pavement/SlantedAndEnchan",
		},
		"status with color": {
			columnName: "status",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Status: status.CurrentStatus,
				},
			},
			columnWidth:    10,
			expectedOutput: "\x1b[32mCurrent\x1b[0m",
		},
		"status trimmed": {
			columnName: "status",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Status: status.NotFoundStatus,
				},
			},
			columnWidth:    5,
			expectedOutput: "NotFo",
		},
		"conditions with color": {
			columnName: "conditions",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Resource: mustResourceWithConditions([]condition{
						{
							Type:   "Ready",
							Status: v1.ConditionUnknown,
						},
					}),
				},
			},
			columnWidth:    10,
			expectedOutput: "\x1b[33mReady\x1b[0m",
		},
		"conditions trimmed": {
			columnName: "conditions",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Resource: mustResourceWithConditions([]condition{
						{
							Type:   "Ready",
							Status: v1.ConditionTrue,
						},
						{
							Type:   "Reconciling",
							Status: v1.ConditionFalse,
						},
					}),
				},
			},
			columnWidth:    10,
			expectedOutput: "\x1b[32mReady\x1b[0m,\x1b[31mReco\x1b[0m",
		},
		"age not found": {
			columnName: "age",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Resource: &unstructured.Unstructured{},
				},
			},
			columnWidth:    10,
			expectedOutput: "-",
		},
		"age": {
			columnName: "age",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Resource: mustResourceWithCreationTimestamp(45 * time.Minute),
				},
			},
			columnWidth:    10,
			expectedOutput: "45m",
		},
		"message without error": {
			columnName: "message",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Message: "this is a test",
				},
			},
			columnWidth:    30,
			expectedOutput: "this is a test",
		},
		"message from error": {
			columnName: "message",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Message: "this is a test",
					Error:   fmt.Errorf("something went wrong somewhere"),
				},
			},
			columnWidth:    50,
			expectedOutput: "something went wrong somewhere",
		},
		"message trimmed": {
			columnName: "message",
			resource: &fakeResource{
				resourceStatus: &pe.ResourceStatus{
					Message: "this is a test",
				},
			},
			columnWidth:    6,
			expectedOutput: "this i",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			columnDef := MustColumn(tc.columnName)

			var buf bytes.Buffer
			_, err := columnDef.PrintResource(&buf, tc.columnWidth, tc.resource)
			if err != nil {
				t.Error(err)
			}

			if want, got := tc.expectedOutput, buf.String(); want != got {
				t.Errorf("expected %q, but got %q", want, got)
			}
		})
	}
}

type condition struct {
	Type   string
	Status v1.ConditionStatus
}

func mustResourceWithConditions(conditions []condition) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: make(map[string]interface{}),
	}
	var conditionsSlice []interface{}
	for _, c := range conditions {
		cond := make(map[string]interface{})
		cond["type"] = c.Type
		cond["status"] = string(c.Status)
		conditionsSlice = append(conditionsSlice, cond)
	}
	err := unstructured.SetNestedSlice(u.Object, conditionsSlice,
		"status", "conditions")
	if err != nil {
		panic(err)
	}
	return u
}

func mustResourceWithCreationTimestamp(age time.Duration) *unstructured.Unstructured {
	u := &unstructured.Unstructured{
		Object: make(map[string]interface{}),
	}
	creationTime := time.Now().Add(-age)
	u.SetCreationTimestamp(metav1.Time{
		Time: creationTime,
	})
	return u
}
