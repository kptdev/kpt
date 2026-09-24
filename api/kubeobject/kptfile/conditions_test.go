// Copyright 2026 The kpt Authors
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

package kptfile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kptfileapi "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/api/kubeobject"
)

const (
	ConditionTypeReady          = "Ready"
	ConditionTypeRendered       = "Rendered"
	ConditionTypeRenderFinished = "RenderFinished"

	reason  = "Reason"
	message = "Message"
)

func newKptfileWithConditions(t *testing.T, conds ...kptfileapi.Condition) *KptfileKubeObject {
	kf := newEmptyKptfile(t)

	var objs kubeobject.SliceSubObjects
	for _, c := range conds {
		ko, err := kubeobject.NewFromTypedObject(c)
		require.NoError(t, err)
		objs = append(objs, &ko.SubObject)
	}
	_ = kf.SetConditions(objs)
	return kf
}

func TestIsType(t *testing.T) {
	cond := kptfileapi.Condition{Type: ConditionTypeReady}
	ko, err := kubeobject.NewFromTypedObject(cond)
	require.NoError(t, err)

	assert.True(t, IsType(ConditionTypeReady)(&ko.SubObject))
	assert.False(t, IsType(ConditionTypeRendered)(&ko.SubObject))
}

func TestIsTypeAndStatus(t *testing.T) {
	cond := kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue}
	ko, err := kubeobject.NewFromTypedObject(cond)
	require.NoError(t, err)

	assert.True(t, IsTypeAndStatus(ConditionTypeReady, kptfileapi.ConditionTrue)(&ko.SubObject))
	assert.False(t, IsTypeAndStatus(ConditionTypeReady, kptfileapi.ConditionFalse)(&ko.SubObject))
	assert.False(t, IsTypeAndStatus(ConditionTypeRendered, kptfileapi.ConditionTrue)(&ko.SubObject))
}

func TestIsConditionType(t *testing.T) {
	ko, err := kubeobject.NewFromTypedObject(map[string]any{"conditionType": ConditionTypeReady})
	require.NoError(t, err)

	assert.True(t, IsConditionType(ConditionTypeReady)(&ko.SubObject))
	assert.False(t, IsConditionType(ConditionTypeRendered)(&ko.SubObject))
}

func TestSetConditions(t *testing.T) {
	kf := newEmptyKptfile(t)
	conds := []kptfileapi.Condition{
		{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
		{Type: ConditionTypeRendered, Status: kptfileapi.ConditionFalse},
	}

	var objs kubeobject.SliceSubObjects
	for _, c := range conds {
		ko, err := kubeobject.NewFromTypedObject(c)
		require.NoError(t, err)
		objs = append(objs, &ko.SubObject)
	}

	err := kf.SetConditions(objs)
	require.NoError(t, err)

	got := kf.Conditions()
	assert.Len(t, got, 2)
}

func TestGetCondition(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
	)

	t.Run("Exists", func(t *testing.T) {
		cond := kf.GetCondition(ConditionTypeReady)
		require.NotNil(t, cond)
		assert.Equal(t, ConditionTypeReady, cond.GetString("type"))
		assert.Equal(t, kptfileapi.ConditionTrue, kptfileapi.ConditionStatus(cond.GetString("status")))
	})

	t.Run("Doesn't exist", func(t *testing.T) {
		cond := kf.GetCondition(ConditionTypeRendered)
		assert.Nil(t, cond)
	})
}

func TestIsStatusConditionPresentAndEqual(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
	)

	assert.True(t, kf.IsStatusConditionPresentAndEqual(ConditionTypeReady, kptfileapi.ConditionTrue))
	assert.False(t, kf.IsStatusConditionPresentAndEqual(ConditionTypeReady, kptfileapi.ConditionFalse))
	assert.False(t, kf.IsStatusConditionPresentAndEqual(ConditionTypeRendered, kptfileapi.ConditionTrue))
}

func TestIsStatusCondition(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
		kptfileapi.Condition{Type: ConditionTypeRendered, Status: kptfileapi.ConditionFalse},
	)

	t.Run("True", func(t *testing.T) {
		assert.True(t, kf.IsStatusConditionTrue(ConditionTypeReady))
		assert.False(t, kf.IsStatusConditionTrue(ConditionTypeRendered))
		assert.False(t, kf.IsStatusConditionTrue(ConditionTypeRenderFinished))
	})

	t.Run("False", func(t *testing.T) {
		assert.True(t, kf.IsStatusConditionFalse(ConditionTypeRendered))
		assert.False(t, kf.IsStatusConditionFalse(ConditionTypeReady))
		assert.False(t, kf.IsStatusConditionFalse(ConditionTypeRenderFinished))
	})
}

func TestGetTypedCondition(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue, Reason: reason, Message: message},
	)

	t.Run("Exists", func(t *testing.T) {
		cond, err := kf.GetTypedCondition(ConditionTypeReady)
		require.NoError(t, err)

		assert.Equal(t, ConditionTypeReady, cond.Type)
		assert.Equal(t, kptfileapi.ConditionTrue, cond.Status)
		assert.Equal(t, reason, cond.Reason)
		assert.Equal(t, message, cond.Message)
	})

	t.Run("Doesn't exist", func(t *testing.T) {
		cond, err := kf.GetTypedCondition(ConditionTypeRendered)
		require.NoError(t, err)
		assert.Empty(t, cond.Type)
	})
}

func TestSetTypedCondition(t *testing.T) {
	kf := newEmptyKptfile(t)

	t.Run("Add", func(t *testing.T) {
		cond := kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue, Reason: reason, Message: message}

		err := kf.SetTypedCondition(cond)
		require.NoError(t, err)

		got := kf.GetCondition(ConditionTypeReady)
		require.NotNil(t, got)
		assert.Equal(t, ConditionTypeReady, got.GetString("type"))
		assert.Equal(t, string(kptfileapi.ConditionTrue), got.GetString("status"))
		assert.Equal(t, reason, got.GetString("reason"))
		assert.Equal(t, message, got.GetString("message"))
	})

	t.Run("Update", func(t *testing.T) {
		const (
			reason2  = reason + "2"
			message2 = message + "2"
		)
		cond2 := kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionFalse, Reason: reason2, Message: message2}

		err := kf.SetTypedCondition(cond2)
		require.NoError(t, err)

		got := kf.GetCondition(ConditionTypeReady)
		require.NotNil(t, got)
		assert.Equal(t, string(kptfileapi.ConditionFalse), got.GetString("status"))
		assert.Equal(t, reason2, got.GetString("reason"))
		assert.Equal(t, message2, got.GetString("message"))
	})
}

func TestApplyDefaultCondition(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
	)

	t.Run("Add", func(t *testing.T) {
		err := kf.ApplyDefaultCondition(kptfileapi.Condition{Type: ConditionTypeRendered, Status: kptfileapi.ConditionTrue})
		require.NoError(t, err)
		assert.Len(t, kf.Conditions(), 2)
	})

	t.Run("No duplicate", func(t *testing.T) {
		lenBefore := len(kf.Conditions())
		err := kf.ApplyDefaultCondition(kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionFalse})
		require.NoError(t, err)
		assert.Len(t, kf.Conditions(), lenBefore)
	})
}

func TestDeleteConditionByType(t *testing.T) {
	kf := newKptfileWithConditions(t,
		kptfileapi.Condition{Type: ConditionTypeReady, Status: kptfileapi.ConditionTrue},
		kptfileapi.Condition{Type: ConditionTypeRendered, Status: kptfileapi.ConditionFalse},
	)

	t.Run("Successful", func(t *testing.T) {
		err := kf.DeleteConditionByType(ConditionTypeReady)
		require.NoError(t, err)
		require.Len(t, kf.Conditions(), 1)
		assert.Equal(t, ConditionTypeRendered, kf.Conditions()[0].GetString("type"))
	})

	t.Run("Non-existent", func(t *testing.T) {
		lenBefore := len(kf.Conditions())
		err := kf.DeleteConditionByType(ConditionTypeRenderFinished)
		require.NoError(t, err)
		assert.Len(t, kf.Conditions(), lenBefore)
	})
}

func TestReadinessGates(t *testing.T) {
	kf := newEmptyKptfile(t)
	gates := []kptfileapi.ReadinessGate{
		{ConditionType: ConditionTypeRendered},
		{ConditionType: ConditionTypeRenderFinished},
	}
	var objs kubeobject.SliceSubObjects
	for _, g := range gates {
		ko, err := kubeobject.NewFromTypedObject(g)
		require.NoError(t, err)
		objs = append(objs, &ko.SubObject)
	}

	err := kf.SetReadinessGates(objs)
	require.NoError(t, err)

	got := kf.ReadinessGates()
	require.Len(t, got, 2)
}

func TestEnsureReadinessGates(t *testing.T) {
	kf := newEmptyKptfile(t)

	t.Run("Add", func(t *testing.T) {
		err := kf.EnsureReadinessGates([]kptfileapi.ReadinessGate{{ConditionType: ConditionTypeRendered}})
		require.NoError(t, err)
		require.Len(t, kf.ReadinessGates(), 1)
		assert.Equal(t, ConditionTypeRendered, kf.ReadinessGates()[0].GetString("conditionType"))

		err = kf.EnsureReadinessGates([]kptfileapi.ReadinessGate{{ConditionType: ConditionTypeRenderFinished}})
		require.NoError(t, err)
		assert.Len(t, kf.ReadinessGates(), 2)
	})

	t.Run("No duplicate", func(t *testing.T) {
		err := kf.EnsureReadinessGates([]kptfileapi.ReadinessGate{
			{ConditionType: ConditionTypeRendered},
			{ConditionType: ConditionTypeRenderFinished},
		})
		require.NoError(t, err)
		require.Len(t, kf.ReadinessGates(), 2)

		err = kf.EnsureReadinessGates([]kptfileapi.ReadinessGate{{ConditionType: ConditionTypeRenderFinished}})
		require.NoError(t, err)
		assert.Len(t, kf.ReadinessGates(), 2)
	})
}
