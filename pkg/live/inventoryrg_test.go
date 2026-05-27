// Copyright 2020,2026 The kpt Authors
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

package live

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"sigs.k8s.io/cli-utils/pkg/apis/actuation"
	"sigs.k8s.io/cli-utils/pkg/common"
	"sigs.k8s.io/cli-utils/pkg/inventory"
	"sigs.k8s.io/cli-utils/pkg/object"
)

var testNamespace = "test-inventory-namespace"
var inventoryObjName = "test-inventory-obj"
var testInventoryLabel = "test-inventory-label"

var inventoryObj = &unstructured.Unstructured{
	Object: map[string]any{
		"apiVersion": "kpt.dev/v1alpha1",
		"kind":       "ResourceGroup",
		"metadata": map[string]any{
			"name":      inventoryObjName,
			"namespace": testNamespace,
			"labels": map[string]any{
				common.InventoryLabel: testInventoryLabel,
			},
		},
		"spec": map[string]any{
			"resources": []any{},
		},
	},
}

var testDeployment = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-deployment",
	GroupKind: schema.GroupKind{
		Group: "apps",
		Kind:  "Deployment",
	},
}

var testService = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-deployment",
	GroupKind: schema.GroupKind{
		Group: "apps",
		Kind:  "Service",
	},
}

var testPod = object.ObjMetadata{
	Namespace: testNamespace,
	Name:      "test-pod",
	GroupKind: schema.GroupKind{
		Group: "",
		Kind:  "Pod",
	},
}

func TestLoadStore(t *testing.T) {
	tests := map[string]struct {
		inv       *unstructured.Unstructured
		objs      []object.ObjMetadata
		objStatus []actuation.ObjectStatus
		isError   bool
	}{
		"Nil inventory is error": {
			inv:     nil,
			objs:    []object.ObjMetadata{},
			isError: true,
		},
		"No inventory objects is valid": {
			inv:     inventoryObj,
			objs:    []object.ObjMetadata{},
			isError: false,
		},
		"Simple test": {
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testPod},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testPod),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationPending,
					Reconcile:       actuation.ReconcilePending,
				},
			},
			isError: false,
		},
		"Test two objects": {
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testDeployment, testService},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testDeployment),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testService),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
			},
			isError: false,
		},
		"Test three objects": {
			inv:  inventoryObj,
			objs: []object.ObjMetadata{testDeployment, testService, testPod},
			objStatus: []actuation.ObjectStatus{
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testDeployment),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testService),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationSucceeded,
					Reconcile:       actuation.ReconcileSucceeded,
				},
				{
					ObjectReference: inventory.ObjectReferenceFromObjMetadata(testPod),
					Strategy:        actuation.ActuationStrategyApply,
					Actuation:       actuation.ActuationPending,
					Reconcile:       actuation.ReconcilePending,
				},
			},
			isError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			wrapped := WrapInventoryObj(tc.inv)
			_ = wrapped.Store(tc.objs, tc.objStatus)
			invStored, err := wrapped.GetObject()
			if tc.isError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			wrapped = WrapInventoryObj(invStored)
			objs, err := wrapped.Load()
			require.NoError(t, err)
			require.True(t, objs.Equal(tc.objs), "expected inventory objs (%v), got (%v)", tc.objs, objs)
			resourceStatus, _, err := unstructured.NestedSlice(invStored.Object, "status", "resourceStatuses")
			require.NoError(t, err)
			require.Len(t, resourceStatus, len(tc.objStatus), "expected %d resource status but got %d", len(tc.objStatus), len(resourceStatus))
		})
	}
}

var cmInvObj = &unstructured.Unstructured{
	Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      inventoryObjName,
			"namespace": testNamespace,
			"labels": map[string]any{
				common.InventoryLabel: testInventoryLabel,
			},
		},
	},
}

func TestIsResourceGroupInventory(t *testing.T) {
	tests := map[string]struct {
		invObj   *unstructured.Unstructured
		expected bool
		isError  bool
	}{
		"Nil inventory is error": {
			invObj:   nil,
			expected: false,
			isError:  true,
		},
		"ConfigMap inventory is false": {
			invObj:   cmInvObj,
			expected: false,
			isError:  false,
		},
		"ResourceGroup inventory is false": {
			invObj:   inventoryObj,
			expected: true,
			isError:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			actual, err := IsResourceGroupInventory(tc.invObj)
			if tc.isError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual, "expected inventory as (%t), got (%t)", tc.expected, actual)
		})
	}
}

// TestWrapInventoryObjWithContext_StoresContext stores ctx on the wrapper.
func TestWrapInventoryObjWithContext_StoresContext(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "propagated")

	storage := WrapInventoryObjWithContext(ctx)(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	require.True(t, ok, "WrapInventoryObjWithContext produced unexpected type %T", storage)
	require.NotNil(t, icm.ctx, "expected ctx on InventoryResourceGroup; got nil")
	require.Equal(t, "propagated", icm.ctx.Value(ctxKey{}), "expected stored ctx to carry propagated value")
}

// TestWrapInventoryObj_LeavesContextNil keeps legacy nil ctx.
func TestWrapInventoryObj_LeavesContextNil(t *testing.T) {
	storage := WrapInventoryObj(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	require.True(t, ok, "WrapInventoryObj produced unexpected type %T", storage)
	require.Nil(t, icm.ctx, "expected legacy wrapper to leave ctx nil; got %v", icm.ctx)
}

// TestWrapInventoryObjWithContext_NilCtxDefaultsToBackground normalizes nil ctx.
func TestWrapInventoryObjWithContext_NilCtxDefaultsToBackground(t *testing.T) {
	//nolint:staticcheck // SA1012: deliberately passing a nil context to exercise the nil-safety guard.
	storage := WrapInventoryObjWithContext(nil)(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	require.True(t, ok, "WrapInventoryObjWithContext(nil) produced unexpected type %T", storage)
	require.NotNil(t, icm.ctx, "expected nil ctx to be normalized to Background(); got nil")
	// Background() never cancels; Done() returns a nil channel.
	require.Nil(t, icm.ctx.Done(), "expected Background()-equivalent ctx; Done() returned non-nil")
}

// TestResourceGroupCRDMatched_BackCompatSignaturePreserved pins exported signatures.
func TestResourceGroupCRDMatched_BackCompatSignaturePreserved(t *testing.T) {
	pinSignatures := func(
		_ func(cmdutil.Factory) bool,
		_ func(context.Context, cmdutil.Factory) bool,
	) {
	}
	pinSignatures(ResourceGroupCRDMatched, ResourceGroupCRDMatchedWithContext)
}

// TestContextOrBackground covers stored ctx and fallback behavior.
func TestContextOrBackground(t *testing.T) {
	t.Run("returns stored ctx when set", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		icm := &InventoryResourceGroup{ctx: ctx}

		got := icm.contextOrBackground()
		require.Same(t, ctx, got, "expected contextOrBackground to return the stored ctx")
		// Cancellation on the original ctx must be visible through the returned ctx.
		cancel()
		select {
		case <-got.Done():
		default:
			require.FailNow(t, "returned ctx did not observe cancellation of the stored ctx")
		}
	})

	t.Run("falls back to Background when nil", func(t *testing.T) {
		icm := &InventoryResourceGroup{}
		got := icm.contextOrBackground()
		require.NotNil(t, got, "contextOrBackground returned nil; expected context.Background()")
		// Background() never cancels; Done() returns a nil channel.
		require.Nil(t, got.Done(), "expected Background-equivalent ctx; Done channel was not nil")
	})
}
