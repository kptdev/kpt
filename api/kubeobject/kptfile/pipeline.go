// Copyright 2024-2026 The kpt Authors
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
	"fmt"
	"reflect"
	"slices"

	kptfileapi "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/api/kubeobject"
)

// UpsertMutatorFunctions ensures that the given KRM functions are added to or updated in the Kptfile's mutators list.
// If the function already exists, it is updated. If it doesn't exist, it is added at the specified position.
// If insertPosition is negative, the insert position is counted backwards from the end of the list
// (i.e. -1 will append to the list).
func (kf *KptfileKubeObject) UpsertMutatorFunctions(fns []kptfileapi.Function, insertPosition int) error {
	return kf.UpsertPipelineFunctions(fns, "mutators", insertPosition)
}

// UpsertValidatorFunctions ensures that the given KRM functions are added to or updated in the Kptfile's validators list.
// If the function already exists, it is updated. If it doesn't exist, it is added at the specified position.
// If insertPosition is negative, the insert position is counted backwards from the end of the list
// (i.e. -1 will append to the list).
func (kf *KptfileKubeObject) UpsertValidatorFunctions(fns []kptfileapi.Function, insertPosition int) error {
	return kf.UpsertPipelineFunctions(fns, "validators", insertPosition)
}

// UpsertPipelineFunctions ensures that the given KRM functions are added to or updated in the Kptfile pipeline's given field (`fieldname`).
// `fieldName` should be either "mutators" or "validators".
// If the function already exists, it is updated. If it doesn't exist, it is added at the specified position.
// If insertPosition is negative, the insert position is counted backwards from the end of the list
// (i.e. -1 will append to the list).
func (kf *KptfileKubeObject) UpsertPipelineFunctions(fns []kptfileapi.Function, fieldName string, insertPosition int) error {
	if len(fns) == 0 {
		return nil
	}
	pipelineKObj := kf.UpsertMap("pipeline")
	fnKObjs, _, _ := pipelineKObj.NestedSlice(fieldName)
	for _, newKrmFn := range fns {
		var err error
		var inserted bool
		fnKObjs, inserted, err = upsertKrmFunction(fnKObjs, newKrmFn, insertPosition)
		if err != nil {
			return err
		}
		if inserted && insertPosition >= 0 {
			// if the function was inserted at the beginning, we need to update the insert position
			// for the next function
			insertPosition++
		}
	}
	return pipelineKObj.SetSlice(fnKObjs, fieldName)
}

// upsertKrmFunction ensures that a KRM function is added or updated in the given list of function objects.
// If the function already exists, it is updated. If it doesn't exist, it is added at the specified position.
// If insertPosition is negative, the insert position is counted backwards from the end of the list
// (i.e. -1 will append to the list).
func upsertKrmFunction(
	fnKObjs kubeobject.SliceSubObjects,
	newKrmFn kptfileapi.Function,
	insertPosition int,
) (kubeobject.SliceSubObjects, bool, error) {
	if newKrmFn.Name == "" {
		// match by content
		fnObj, err := findFunctionByContent(fnKObjs, &newKrmFn)
		if err != nil {
			return nil, false, err
		}
		if fnObj != nil {
			// function already exists, skip to avoid duplicates
			return fnKObjs, false, nil
		}
	} else {
		// match by name
		fnObj := findFunctionByName(fnKObjs, newKrmFn.Name)
		if fnObj != nil {
			// function with the same name exists, update it
			var origKrmFn kptfileapi.Function
			err := fnObj.As(&origKrmFn)
			if err != nil {
				return nil, false, fmt.Errorf("failed to parse KRM function from YAML: %w", err)
			}
			err = fnObj.SetFromTypedObject(newKrmFn)
			if err != nil {
				return nil, false, fmt.Errorf("failed to update KRM function in Kptfile: %w", err)
			}
			return fnKObjs, false, nil
		}
	}

	// function does not exist, insert it
	newFuncObj, err := kubeobject.NewFromTypedObject(newKrmFn)
	if err != nil {
		return nil, false, err
	}
	if insertPosition < 0 {
		insertPosition = len(fnKObjs) + insertPosition + 1
	}
	fnKObjs = slices.Insert(fnKObjs, insertPosition, &newFuncObj.SubObject)
	return fnKObjs, true, nil
}

// findFunction returns with the first KRM function in the list with the given name
func findFunctionByName(haystack kubeobject.SliceSubObjects, name string) *kubeobject.SubObject {
	for _, fnObj := range haystack {
		// match by name
		objName, found, _ := fnObj.NestedString("name")
		if found && objName == name {
			return fnObj
		}
	}
	return nil
}

// findFunctionByContent returns with the first KRM function in the list that matches the content of the needle
func findFunctionByContent(haystack kubeobject.SliceSubObjects, needle *kptfileapi.Function) (*kubeobject.SubObject, error) {
	for _, fnObj := range haystack {
		var krmFn kptfileapi.Function
		err := fnObj.As(&krmFn)
		if err != nil {
			return nil, fmt.Errorf("failed to parse KRM function from YAML: %w", err)
		}
		// ignore diff in name
		krmFn.Name = needle.Name
		if reflect.DeepEqual(krmFn, *needle) {
			return fnObj, nil
		}
	}
	return nil, nil
}
