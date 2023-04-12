// Copyright 2022 The kpt Authors
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
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/engine"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	"sigs.k8s.io/yaml"
)

const (
	ArgoGroup   = "argoproj.io"
	Rollout     = "Rollout"
	Degraded    = "Degraded"
	Failed      = "Failed"
	Healthy     = "Healthy"
	Paused      = "Paused"
	Progressing = "Progressing"
)

type RolloutStatusReader struct {
	Mapper meta.RESTMapper
}

func NewRolloutStatusReader(mapper meta.RESTMapper) engine.StatusReader {
	return &RolloutStatusReader{
		Mapper: mapper,
	}
}

var _ engine.StatusReader = &RolloutStatusReader{}

// Supports returns true for all rollout resources.
func (r *RolloutStatusReader) Supports(gk schema.GroupKind) bool {
	return gk.Group == ArgoGroup && gk.Kind == Rollout
}

func (r *RolloutStatusReader) Compute(u *unstructured.Unstructured) (*status.Result, error) {
	result := status.Result{
		Status:     status.UnknownStatus,
		Message:    status.GetStringField(u.Object, ".status.message", ""),
		Conditions: make([]status.Condition, 0),
	}
	// ensure that the meta generation is observed
	generation, found, err := unstructured.NestedInt64(u.Object, "metadata", "generation")
	if err != nil {
		return &result, fmt.Errorf("looking up metadata.generation from resource: %w", err)
	}
	if !found {
		return &result, fmt.Errorf("metadata.generation not found")
	}

	// Argo Rollouts defines the observedGeneration field in the Rollout object as a string
	// so read it as a string here
	observedGenerationString, found, err := unstructured.NestedString(u.Object, "status", "observedGeneration")
	if err != nil {
		return &result, fmt.Errorf("looking up status.observedGeneration from resource: %w", err)
	}
	if !found {
		// We know that Rollout resources uses the ObservedGeneration pattern, so consider it
		// an error if it is not found.
		return &result, fmt.Errorf("status.ObservedGeneration not found")
	}
	// If no errors detected and the field is found
	// Parse it to become an integer
	observedGeneration, err := strconv.ParseInt(observedGenerationString, 10, 64)
	if err != nil {
		return &result, fmt.Errorf("looking up status.observedGeneration from resource: %w", err)
	}

	if generation != observedGeneration {
		msg := fmt.Sprintf("%s generation is %d, but latest observed generation is %d", u.GetKind(), generation, observedGeneration)
		result.Status = status.InProgressStatus
		result.Message = msg
		return &result, nil
	}

	phase, phaseFound, err := unstructured.NestedString(u.Object, "status", "phase")
	if err != nil {
		return &result, fmt.Errorf("looking up status.phase from resource: %w", err)
	}
	if !phaseFound {
		// We know that Rollout resources uses the phase pattern, so consider it
		// an error if it is not found.
		return &result, fmt.Errorf("status.phase not found")
	}

	conditions, condFound, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil {
		return &result, fmt.Errorf("looking up status.conditions from resource: %w", err)
	}
	if condFound {
		data, err := yaml.Marshal(conditions)
		if err != nil {
			return &result, fmt.Errorf("failed to marshal conditions for %s/%s", u.GetNamespace(), u.GetName())
		}
		err = yaml.Unmarshal(data, &result.Conditions)
		if err != nil {
			return &result, fmt.Errorf("failed to unmarshal conditions for %s/%s", u.GetNamespace(), u.GetName())
		}
	}

	specReplicas := status.GetIntField(u.Object, ".spec.replicas", 1) // Controller uses 1 as default if not specified.
	statusReplicas := status.GetIntField(u.Object, ".status.replicas", 0)
	updatedReplicas := status.GetIntField(u.Object, ".status.updatedReplicas", 0)
	readyReplicas := status.GetIntField(u.Object, ".status.readyReplicas", 0)
	availableReplicas := status.GetIntField(u.Object, ".status.availableReplicas", 0)

	if specReplicas > statusReplicas {
		message := fmt.Sprintf("replicas: %d/%d", statusReplicas, specReplicas)
		result.Status = status.InProgressStatus
		result.Message = message

		return &result, nil
	}

	if statusReplicas > specReplicas {
		message := fmt.Sprintf("Pending termination: %d", statusReplicas-specReplicas)
		result.Status = status.InProgressStatus
		result.Message = message
		return &result, nil
	}

	if updatedReplicas > availableReplicas {
		message := fmt.Sprintf("Available: %d/%d", availableReplicas, updatedReplicas)
		result.Status = status.InProgressStatus
		result.Message = message
		return &result, nil
	}

	if specReplicas > readyReplicas {
		message := fmt.Sprintf("Ready: %d/%d", readyReplicas, specReplicas)
		result.Status = status.InProgressStatus
		result.Message = message
		return &result, nil
	}

	message := status.GetStringField(u.Object, ".status.message", "")
	if message != "" {
		message += " "
	}
	message += fmt.Sprintf("Ready Replicas: %d, Updated Replicas: %d", readyReplicas, updatedReplicas)
	result.Message = message

	switch phase {
	case Degraded, Failed:
		result.Status = status.FailedStatus
	case Healthy:
		result.Status = status.CurrentStatus
	case Paused, Progressing:
		result.Status = status.InProgressStatus
	default:
		// Undefined status
		result.Status = status.UnknownStatus
	}
	return &result, nil
}

func (r *RolloutStatusReader) ReadStatus(ctx context.Context, reader engine.ClusterReader, id object.ObjMetadata) (
	*event.ResourceStatus, error) {
	gvk, err := toGVK(id.GroupKind, r.Mapper)
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

	return r.ReadStatusForObject(ctx, reader, &u)
}

func (r *RolloutStatusReader) ReadStatusForObject(_ context.Context, _ engine.ClusterReader, u *unstructured.Unstructured) (
	*event.ResourceStatus, error) {
	id := object.UnstructuredToObjMetadata(u)

	// First check if the resource is in the process of being deleted.
	deletionTimestamp, found, err := unstructured.NestedString(u.Object, "metadata", "deletionTimestamp")
	if err != nil {
		return newUnknownResourceStatus(id, u, err), nil
	}
	if found && deletionTimestamp != "" {
		return newResourceStatus(id, status.TerminatingStatus, u, "Resource scheduled for deletion"), nil
	}

	res, err := r.Compute(u)
	if err != nil {
		return newUnknownResourceStatus(id, u, err), nil
	}

	return newResourceStatus(id, res.Status, u, res.Message), nil
}
