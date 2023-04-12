// Copyright 2021 The kpt Authors
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

package status

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

const (
	ConditionReady = "Ready"
)

// ConfigConnectorStatusReader can compute reconcile status for Config Connector
// resources. It leverages information in the `Reason` field of the `Ready` condition.
// TODO(mortent): Make more of the convencience functions and types from cli-utils
// exported so we can simplify this.
type ConfigConnectorStatusReader struct {
	Mapper meta.RESTMapper
}

func NewConfigConnectorStatusReader(mapper meta.RESTMapper) engine.StatusReader {
	return &ConfigConnectorStatusReader{
		Mapper: mapper,
	}
}

var _ engine.StatusReader = &ConfigConnectorStatusReader{}

// Supports returns true for all Config Connector resources.
func (c *ConfigConnectorStatusReader) Supports(gk schema.GroupKind) bool {
	return strings.HasSuffix(gk.Group, "cnrm.cloud.google.com")
}

func (c *ConfigConnectorStatusReader) ReadStatus(ctx context.Context, reader engine.ClusterReader, id object.ObjMetadata) (*event.ResourceStatus, error) {
	gvk, err := toGVK(id.GroupKind, c.Mapper)
	if err != nil {
		return newUnknownResourceStatus(id, nil, err), nil
	}

	key := types.NamespacedName{
		Name:      id.Name,
		Namespace: id.Namespace,
	}

	var u unstructured.Unstructured
	u.SetGroupVersionKind(gvk)
	err = reader.Get(ctx, key, &u)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		if apierrors.IsNotFound(err) {
			return newResourceStatus(id, status.NotFoundStatus, &u, "Resource not found"), nil
		}
		return newUnknownResourceStatus(id, nil, err), nil
	}

	return c.ReadStatusForObject(ctx, reader, &u)
}

func (c *ConfigConnectorStatusReader) ReadStatusForObject(_ context.Context, _ engine.ClusterReader, u *unstructured.Unstructured) (*event.ResourceStatus, error) {
	id := object.UnstructuredToObjMetadata(u)

	// First check if the resource is in the process of being deleted.
	deletionTimestamp, found, err := unstructured.NestedString(u.Object, "metadata", "deletionTimestamp")
	if err != nil {
		return newUnknownResourceStatus(id, u, err), nil
	}
	if found && deletionTimestamp != "" {
		return newResourceStatus(id, status.TerminatingStatus, u, "Resource scheduled for deletion"), nil
	}

	res, err := c.Compute(u)

	if err != nil {
		return newUnknownResourceStatus(id, u, err), nil
	}
	return newResourceStatus(id, res.Status, u, res.Message), nil
}

func (c *ConfigConnectorStatusReader) Compute(u *unstructured.Unstructured) (*status.Result, error) {
	if u.GroupVersionKind().Kind == "ConfigConnectorContext" {
		return computeStatusForConfigConnectorContext(u)
	}

	return computeStatusForConfigConnector(u)
}

func computeStatusForConfigConnectorContext(u *unstructured.Unstructured) (*status.Result, error) {
	healthy, found, err := unstructured.NestedBool(u.Object, "status", "healthy")
	if err != nil {
		e := fmt.Errorf("looking up status.healthy from resource: %w", err)
		return nil, e
	}
	if !found {
		msg := "status.healthy property not set"
		return &status.Result{
			Status:  status.InProgressStatus,
			Message: msg,
			Conditions: []status.Condition{
				{
					Type:    status.ConditionReconciling,
					Status:  corev1.ConditionTrue,
					Reason:  "NotReady",
					Message: msg,
				},
			},
		}, nil
	}
	if !healthy {
		msg := "status.healthy is false"
		return &status.Result{
			Status:  status.InProgressStatus,
			Message: msg,
			Conditions: []status.Condition{
				{
					Type:    status.ConditionReconciling,
					Status:  corev1.ConditionTrue,
					Reason:  "NotReady",
					Message: msg,
				},
			},
		}, nil
	}
	return &status.Result{
		Status:  status.CurrentStatus,
		Message: "status.healthy is true",
	}, nil
}

func computeStatusForConfigConnector(u *unstructured.Unstructured) (*status.Result, error) {
	// ensure that the meta generation is observed
	generation, found, err := unstructured.NestedInt64(u.Object, "metadata", "generation")
	if err != nil {
		e := fmt.Errorf("looking up metadata.generation from resource: %w", err)
		return nil, e
	}
	if !found {
		e := fmt.Errorf("metadata.generation not found")
		return nil, e
	}

	observedGeneration, found, err := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	if err != nil {
		e := fmt.Errorf("looking up status.observedGeneration from resource: %w", err)
		return nil, e
	}
	if !found {
		// We know that Config Connector resources uses the ObservedGeneration pattern, so consider it
		// an error if it is not found.
		e := fmt.Errorf("status.ObservedGeneration not found")
		return nil, e
	}
	if generation != observedGeneration {
		msg := fmt.Sprintf("%s generation is %d, but latest observed generation is %d", u.GetKind(), generation, observedGeneration)
		return &status.Result{
			Status:  status.InProgressStatus,
			Message: msg,
			Conditions: []status.Condition{
				{
					Type:    status.ConditionReconciling,
					Status:  corev1.ConditionTrue,
					Reason:  "LatestGenerationNotObserved",
					Message: msg,
				},
			},
		}, nil
	}

	obj, err := status.GetObjectWithConditions(u.Object)
	if err != nil {
		return nil, err
	}

	var readyCond status.BasicCondition
	foundCond := false
	for i := range obj.Status.Conditions {
		if obj.Status.Conditions[i].Type == ConditionReady {
			readyCond = obj.Status.Conditions[i]
			foundCond = true
		}
	}

	if !foundCond {
		msg := "Ready condition not set"
		return &status.Result{
			Status:  status.InProgressStatus,
			Message: msg,
			Conditions: []status.Condition{
				{
					Type:    status.ConditionReconciling,
					Status:  corev1.ConditionTrue,
					Reason:  "NoReadyCondition",
					Message: msg,
				},
			},
		}, nil
	}

	if readyCond.Status == corev1.ConditionTrue {
		return &status.Result{
			Status:  status.CurrentStatus,
			Message: "Resource is Current",
		}, nil
	}

	switch readyCond.Reason {
	case "ManagementConflict", "UpdateFailed", "DeleteFailed", "DependencyInvalid":
		return &status.Result{
			Status:  status.FailedStatus,
			Message: readyCond.Message,
			Conditions: []status.Condition{
				{
					Type:    status.ConditionStalled,
					Status:  corev1.ConditionTrue,
					Reason:  readyCond.Reason,
					Message: readyCond.Message,
				},
			},
		}, nil
	}

	return &status.Result{
		Status:  status.InProgressStatus,
		Message: readyCond.Message,
		Conditions: []status.Condition{
			{
				Type:    status.ConditionReconciling,
				Status:  corev1.ConditionTrue,
				Reason:  readyCond.Reason,
				Message: readyCond.Message,
			},
		},
	}, nil
}

func toGVK(gk schema.GroupKind, mapper meta.RESTMapper) (schema.GroupVersionKind, error) {
	mapping, err := mapper.RESTMapping(gk)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	return mapping.GroupVersionKind, nil
}

func newResourceStatus(id object.ObjMetadata, s status.Status, u *unstructured.Unstructured, msg string) *event.ResourceStatus {
	return &event.ResourceStatus{
		Identifier: id,
		Status:     s,
		Resource:   u,
		Message:    msg,
	}
}

func newUnknownResourceStatus(id object.ObjMetadata, u *unstructured.Unstructured, err error) *event.ResourceStatus {
	return &event.ResourceStatus{
		Identifier: id,
		Status:     status.UnknownStatus,
		Error:      err,
		Resource:   u,
	}
}
