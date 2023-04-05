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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/testutil"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	fakemapper "sigs.k8s.io/cli-utils/pkg/testutil"
)

func TestRolloutSupports(t *testing.T) {
	testCases := map[string]struct {
		gk       schema.GroupKind
		supports bool
	}{
		"matches rollout group": {
			gk: schema.GroupKind{
				Group: "argoproj.io",
				Kind:  "Rollout",
			},
			supports: true,
		},
		"doesn't match other resources": {
			gk: schema.GroupKind{
				Group: "apps",
				Kind:  "StatefulSet",
			},
			supports: false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fakeMapper := fakemapper.NewFakeRESTMapper()

			statusReader := &RolloutStatusReader{
				Mapper: fakeMapper,
			}

			supports := statusReader.Supports(tc.gk)

			assert.Equal(t, tc.supports, supports)
		})
	}
}

func TestRolloutReadStatus(t *testing.T) {
	testCases := map[string]struct {
		resource       string
		gvk            schema.GroupVersionKind
		expectedStatus status.Status
		expectedMsg    string
	}{
		"Resource where observedGeneration doesn't match generation is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
status:
  observedGeneration: "41"
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Version: "v1alpha1",
				Kind:    "Rollout",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Rollout generation is 42, but latest observed generation is 41",
		},
		"Resource when spec.replicas more than status.replicas is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 1
  phase: Progressing
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "replicas: 1/2",
		},
		"Resource when status.replicas more than spec.replicas is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 1
status:
  observedGeneration: "42"
  replicas: 2
  phase: Progressing
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Pending termination: 1",
		},
		"Resource when status.updatedReplicas more than spec.availableReplicas is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 1
  phase: Progressing
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Available: 1/2",
		},
		"Resource when spec.replicas more than spec.readyReplicas is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 1
  phase: Progressing
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Ready: 1/2",
		},
		"Resource when status.phase is Degraded is Failed": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 2
  phase: Degraded
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.FailedStatus,
			expectedMsg:    "Ready Replicas: 2, Updated Replicas: 2",
		},
		"Resource when status.phase is Failed is Failed": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 2
  phase: Failed
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.FailedStatus,
			expectedMsg:    "Ready Replicas: 2, Updated Replicas: 2",
		},
		"Resource when status.phase is Healthy is Current": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 2
  phase: Healthy
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.CurrentStatus,
			expectedMsg:    "Ready Replicas: 2, Updated Replicas: 2",
		},
		"Resource when status.phase is Paused is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 2
  phase: Paused
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Ready Replicas: 2, Updated Replicas: 2",
		},
		"Resource when status.phase is Progressing is InProgress": {
			resource: `
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
spec:
  replicas: 2
status:
  observedGeneration: "42"
  replicas: 2
  updatedReplicas: 2
  availableReplicas: 2
  readyReplicas: 2
  phase: Progressing
`,
			gvk: schema.GroupVersionKind{
				Group:   "argoproj.io",
				Kind:    "Rollout",
				Version: "v1alpha1",
			},
			expectedStatus: status.InProgressStatus,
			expectedMsg:    "Ready Replicas: 2, Updated Replicas: 2",
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			obj := testutil.YamlToUnstructured(t, tc.resource)

			fakeClusterReader := &fakeClusterReader{
				getResource: obj,
			}

			fakeMapper := fakemapper.NewFakeRESTMapper(tc.gvk)
			statusReader := &RolloutStatusReader{
				Mapper: fakeMapper,
			}

			res, err := statusReader.ReadStatus(context.Background(), fakeClusterReader, object.UnstructuredToObjMetadata(obj))
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, res.Status)
			assert.Equal(t, tc.expectedMsg, res.Message)
		})
	}
}
