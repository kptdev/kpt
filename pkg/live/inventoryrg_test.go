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
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			wrapped = WrapInventoryObj(invStored)
			objs, err := wrapped.Load()
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			if !objs.Equal(tc.objs) {
				t.Fatalf("expected inventory objs (%v), got (%v)", tc.objs, objs)
			}
			resourceStatus, _, err := unstructured.NestedSlice(invStored.Object, "status", "resourceStatuses")
			if err != nil {
				t.Fatalf("unexpected error %v received", err)
			}
			if len(resourceStatus) != len(tc.objStatus) {
				t.Fatalf("expected %d resource status but got %d", len(tc.objStatus), len(resourceStatus))
			}
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
				if err == nil {
					t.Fatalf("expected error but received none")
				}
				return
			}
			if !tc.isError && err != nil {
				t.Fatalf("unexpected error %v received", err)
				return
			}
			if tc.expected != actual {
				t.Errorf("expected inventory as (%t), got (%t)", tc.expected, actual)
			}
		})
	}
}

// TestWrapInventoryObjWithContext_StoresContext proves the new factory
// threads the caller's context into the InventoryResourceGroup struct.
// This is the mechanism the Apply / ApplyWithPrune methods use to honor
// Ctrl-C / caller timeouts instead of the old context.TODO() behavior.
func TestWrapInventoryObjWithContext_StoresContext(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "propagated")

	storage := WrapInventoryObjWithContext(ctx)(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	if !ok {
		t.Fatalf("WrapInventoryObjWithContext produced unexpected type %T", storage)
	}
	if icm.ctx == nil {
		t.Fatal("expected ctx on InventoryResourceGroup; got nil")
	}
	if got := icm.ctx.Value(ctxKey{}); got != "propagated" {
		t.Fatalf("expected stored ctx to carry propagated value; got %v", got)
	}
}

// TestWrapInventoryObj_LeavesContextNil confirms the legacy wrapper keeps
// ctx nil so contextOrBackground falls back to context.Background() —
// preserving the pre-refactor behavior for callers that haven't migrated.
func TestWrapInventoryObj_LeavesContextNil(t *testing.T) {
	storage := WrapInventoryObj(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	if !ok {
		t.Fatalf("WrapInventoryObj produced unexpected type %T", storage)
	}
	if icm.ctx != nil {
		t.Fatalf("expected legacy wrapper to leave ctx nil; got %v", icm.ctx)
	}
}

// TestWrapInventoryObjWithContext_NilCtxDefaultsToBackground proves the
// factory is nil-safe: passing a nil ctx cannot produce a wrapper that
// would nil-deref inside client-go. The stored ctx is normalized to
// context.Background() at factory construction time.
func TestWrapInventoryObjWithContext_NilCtxDefaultsToBackground(t *testing.T) {
	//nolint:staticcheck // SA1012: deliberately passing a nil context to exercise the nil-safety guard.
	storage := WrapInventoryObjWithContext(nil)(inventoryObj)
	icm, ok := storage.(*InventoryResourceGroup)
	if !ok {
		t.Fatalf("WrapInventoryObjWithContext(nil) produced unexpected type %T", storage)
	}
	if icm.ctx == nil {
		t.Fatal("expected nil ctx to be normalized to Background(); got nil")
	}
	// Background() never cancels; Done() returns a nil channel.
	if icm.ctx.Done() != nil {
		t.Fatalf("expected Background()-equivalent ctx; Done() returned non-nil")
	}
}

// TestResourceGroupCRDMatched_BackCompatSignaturePreserved is a
// compile-time guard that the legacy ResourceGroupCRDMatched(factory)
// signature is still exported, alongside the new context-aware
// ResourceGroupCRDMatchedWithContext(ctx, factory). If either function
// is renamed, removed, or has its signature changed, this test stops
// compiling and the API-compat break is visible immediately.
//
// Uses typed anonymous-function parameters so the compiler verifies
// signature assignability. This pattern is deliberate — staticcheck's
// QF1011 would otherwise suggest removing a `var _ T = fn` type
// annotation, which would silently destroy the guarantee.
//
// We don't invoke the functions because both require a live
// cmdutil.Factory; their runtime behavior is exercised by the
// apply/destroy e2e tests.
func TestResourceGroupCRDMatched_BackCompatSignaturePreserved(t *testing.T) {
	pinSignatures := func(
		_ func(cmdutil.Factory) bool,
		_ func(context.Context, cmdutil.Factory) bool,
	) {
	}
	pinSignatures(ResourceGroupCRDMatched, ResourceGroupCRDMatchedWithContext)
}

// TestContextOrBackground covers both the override path (caller-supplied
// ctx is returned verbatim, including cancellation state) and the
// fallback path (nil ctx becomes context.Background()).
func TestContextOrBackground(t *testing.T) {
	t.Run("returns stored ctx when set", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		icm := &InventoryResourceGroup{ctx: ctx}

		got := icm.contextOrBackground()
		if got != ctx {
			t.Fatalf("expected contextOrBackground to return the stored ctx")
		}
		// Cancellation on the original ctx must be visible through the
		// returned ctx — proof the value isn't copied or unwrapped.
		cancel()
		select {
		case <-got.Done():
			// expected
		default:
			t.Fatalf("returned ctx did not observe cancellation of the stored ctx")
		}
	})

	t.Run("falls back to Background when nil", func(t *testing.T) {
		icm := &InventoryResourceGroup{}
		got := icm.contextOrBackground()
		if got == nil {
			t.Fatal("contextOrBackground returned nil; expected context.Background()")
		}
		// Background() never cancels; Done() returns a nil channel.
		if got.Done() != nil {
			t.Fatalf("expected Background-equivalent ctx; Done channel was not nil")
		}
	})
}
