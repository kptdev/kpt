// Copyright 2023 The kpt Authors
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

package plan

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

// Planner holds the core planning logic and state.
type Planner struct {
}

// BuildPlan is the entry point for building a plan.
func (r *Planner) BuildPlan(ctx context.Context, objects []*unstructured.Unstructured, target *ClusterTarget) (*Plan, error) {
	plan := &Plan{}

	plan.APIVersion = "plan.kpt.dev/v1alpha1"
	plan.Kind = "Plan"

	// TODO: prefetch & invalidate REST Mappings here, so we know they are valid later?

	force := true
	applyOptions := metav1.PatchOptions{
		FieldManager: "kpt/plan",
		Force:        &force,
		DryRun:       []string{"All"},
	}

	for _, object := range objects {
		gvk := object.GroupVersionKind()

		var action Action
		action.Namespace = object.GetNamespace()
		action.Name = object.GetName()
		action.Kind = gvk.Kind
		action.APIVersion = object.GetAPIVersion()
		action.Object = object

		id := gvk.Kind + ":" + action.Namespace + "/" + action.Name

		resource, err := target.ResourceForGVK(ctx, gvk)
		if err != nil {
			// The Kind doesn't even exist; the object can't exist already
			// TODO: We should invalide mappings above, in case our cache is out of date
			action.Type = "Create"
			plan.Spec.Actions = append(plan.Spec.Actions, action)
			continue
		}

		beforeApply, err := resource.Get(ctx, object, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(unwrap(err)) {
				action.Type = "Create"
				plan.Spec.Actions = append(plan.Spec.Actions, action)
				continue
			}
			return nil, fmt.Errorf("failed to read object %s: %w", id, err)
		}

		klog.V(5).Infof("applying %s...", id)
		afterApply, err := resource.Apply(ctx, object, applyOptions)
		if err == nil {
			klog.V(4).Infof("applied OK %s", id)

			// Ignore changes in managed fields, since they are not relevant to the user
			beforeApply.SetManagedFields(nil)
			afterApply.SetManagedFields(nil)

			if reflect.DeepEqual(beforeApply, afterApply) {
				action.Type = "Unchanged"
				plan.Spec.Actions = append(plan.Spec.Actions, action)
			} else {
				diff := cmp.Diff(beforeApply, afterApply)
				if diff != "" {
					klog.Infof("diff is %v", diff)
				}
				action.Type = "ApplyChanges"
				plan.Spec.Actions = append(plan.Spec.Actions, action)
			}
		}

		if err != nil {
			if action.Type == "" {
				klog.Errorf("unknown error applying (%s) %#v", id, err)
				action.Type = "Error"
			}
			plan.Spec.Actions = append(plan.Spec.Actions, action)
			continue
		}
	}

	return plan, nil
}

func unwrap(err error) error {
	var apiStatusError *apierrors.StatusError
	if errors.As(err, &apiStatusError) {
		return apiStatusError
	}
	return err
}
