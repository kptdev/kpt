// Copyright 2021 Google LLC
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
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// ConfigConnectorStatusReader can compute reconcile status for Config Connector
// resources. It leverages information in the `Reason` field of the `Ready` condition.
// TODO(mortent): Make more of the convencience functions and types from cli-utils
// exported so we can simplify this.
type ConfigConnectorStatusReader struct {
	Mapper meta.RESTMapper
}

// Supports returns true for all Config Connector resources.
func (c *ConfigConnectorStatusReader) Supports(gk schema.GroupKind) bool {
	return strings.HasSuffix(gk.Group, "cnrm.cloud.google.com")
}

func (c *ConfigConnectorStatusReader) ReadStatus(ctx context.Context, reader engine.ClusterReader, id object.ObjMetadata) *event.ResourceStatus {
	gvk, err := toGVK(id.GroupKind, c.Mapper)
	if err != nil {
		return newUnknownResourceStatus(id, nil, err)
	}

	key := types.NamespacedName{
		Name:      id.Name,
		Namespace: id.Namespace,
	}

	var u unstructured.Unstructured
	u.SetGroupVersionKind(gvk)
	err = reader.Get(ctx, key, &u)
	if err != nil {
		return newUnknownResourceStatus(id, nil, err)
	}

	return c.ReadStatusForObject(ctx, reader, &u)
}

func (c *ConfigConnectorStatusReader) ReadStatusForObject(_ context.Context, _ engine.ClusterReader, u *unstructured.Unstructured) *event.ResourceStatus {
	id := object.UnstructuredToObjMetadata(u)

	// First check if the resource is in the process of being deleted.
	deletionTimestamp, found, err := unstructured.NestedString(u.Object, "metadata", "deletionTimestamp")
	if err != nil {
		return newUnknownResourceStatus(id, u, err)
	}
	if found && deletionTimestamp != "" {
		return newResourceStatus(id, status.TerminatingStatus, u, "Resource scheduled for deletion")
	}

	// ensure that the meta generation is observed
	generation, found, err := unstructured.NestedInt64(u.Object, "metadata", "generation")
	if err != nil {
		e := fmt.Errorf("looking up metadata.generation from resource: %w", err)
		return newUnknownResourceStatus(id, u, e)
	}
	if !found {
		e := fmt.Errorf("metadata.generation not found")
		return newUnknownResourceStatus(id, u, e)
	}

	observedGeneration, found, err := unstructured.NestedInt64(u.Object, "status", "observedGeneration")
	if err != nil {
		e := fmt.Errorf("looking up status.observedGeneration from resource: %w", err)
		return newUnknownResourceStatus(id, u, e)
	}
	if !found {
		// We know that Config Connector resources uses the ObservedGeneration pattern, so consider it
		// an error if it is not found.
		e := fmt.Errorf("status.ObservedGeneration not found")
		return newUnknownResourceStatus(id, u, e)
	}
	if generation != observedGeneration {
		msg := fmt.Sprintf("%s generation is %d, but latest observed generation is %d", u.GetKind(), generation, observedGeneration)
		return newResourceStatus(id, status.InProgressStatus, u, msg)
	}

	obj, err := status.GetObjectWithConditions(u.Object)
	if err != nil {
		return newUnknownResourceStatus(id, u, err)
	}

	var readyCond status.BasicCondition
	foundCond := false
	for i := range obj.Status.Conditions {
		if obj.Status.Conditions[i].Type == "Ready" {
			readyCond = obj.Status.Conditions[i]
			foundCond = true
		}
	}

	if !foundCond {
		return newResourceStatus(id, status.InProgressStatus, u, "Ready condition not set")
	}

	if readyCond.Status == v1.ConditionTrue {
		return newResourceStatus(id, status.CurrentStatus, u, "Resource is Current")
	}

	switch readyCond.Reason {
	case "ManagementConflict", "UpdateFailed", "DeleteFailed", "DependencyInvalid":
		return newResourceStatus(id, status.FailedStatus, u, readyCond.Message)
	}

	return newResourceStatus(id, status.InProgressStatus, u, readyCond.Message)
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
