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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/testutil"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
	fakemapper "sigs.k8s.io/cli-utils/pkg/testutil"
)

func TestSupports(t *testing.T) {
	testCases := map[string]struct {
		gk       schema.GroupKind
		supports bool
	}{
		"matches config connector group": {
			gk: schema.GroupKind{
				Group: "sql.cnrm.cloud.google.com",
				Kind:  "SQLDatabase",
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

			statusReader := &ConfigConnectorStatusReader{
				Mapper: fakeMapper,
			}

			supports := statusReader.Supports(tc.gk)

			assert.Equal(t, tc.supports, supports)
		})
	}
}

func TestReadStatus(t *testing.T) {
	testCases := map[string]struct {
		resource       string
		gvk            schema.GroupVersionKind
		expectedStatus status.Status
		deleted        bool
	}{
		"Resource with deletionTimestap is Terminating": {
			resource: `
apiVersion: serviceusage.cnrm.cloud.google.com/v1beta1
kind: Service
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
  deletionTimestamp: "2020-01-09T20:56:25Z"
`,
			gvk: schema.GroupVersionKind{
				Group:   "serviceusage.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "Service",
			},
			expectedStatus: status.TerminatingStatus,
		},
		"Resource where observedGeneration doesn't match generation is InProgress": {
			resource: `
apiVersion: serviceusage.cnrm.cloud.google.com/v1beta1
kind: Service
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
status:
  observedGeneration: 41
  conditions:
  - type: Ready
    status: "False"
    reason: UpdateFailed
    message: "Resource couldn't be updated"
`,
			gvk: schema.GroupVersionKind{
				Group:   "serviceusage.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "Service",
			},
			expectedStatus: status.InProgressStatus,
		},
		"Resource with reason UpdateFailed is Failed": {
			resource: `
apiVersion: serviceusage.cnrm.cloud.google.com/v1beta1
kind: Service
metadata:
  name: pubsub.googleapis.com
  namespace: cnrm
  generation: 42
status:
  observedGeneration: 42
  conditions:
  - type: Ready
    status: "False"
    reason: UpdateFailed
    message: "Resource couldn't be updated"
`,
			gvk: schema.GroupVersionKind{
				Group:   "serviceusage.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "Service",
			},
			expectedStatus: status.FailedStatus,
		},

		"Resource has been deleted": {
			resource: `
apiVersion: storage.cnrm.cloud.google.com/v1beta1
kind: StorageBucket
metadata:
  name: fake-bucket
`,
			gvk: schema.GroupVersionKind{
				Group:   "storage.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "StorageBucket",
			},
			expectedStatus: status.NotFoundStatus,
			deleted:        true,
		},

		"ConfigConnectorContext resource with status.healthy set to true": {
			resource: `
apiVersion: core.cnrm.cloud.google.com/v1beta1
kind: ConfigConnectorContext
metadata:
  name: ccc
  namespace: cnrm
  generation: 42
status:
  healthy: true
`,
			gvk: schema.GroupVersionKind{
				Group:   "core.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "ConfigConnectorContext",
			},
			expectedStatus: status.CurrentStatus,
			deleted:        false,
		},

		"ConfigConnectorContext resource with status.healthy set to false": {
			resource: `
apiVersion: core.cnrm.cloud.google.com/v1beta1
kind: ConfigConnectorContext
metadata:
  name: ccc
  namespace: cnrm
  generation: 42
status:
  healthy: false
`,
			gvk: schema.GroupVersionKind{
				Group:   "core.cnrm.cloud.google.com",
				Version: "v1beta1",
				Kind:    "ConfigConnectorContext",
			},
			expectedStatus: status.InProgressStatus,
			deleted:        false,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			obj := testutil.YamlToUnstructured(t, tc.resource)

			fakeClusterReader := &fakeClusterReader{
				getResource: obj,
			}
			// Return not found error if we want the resource to be deleted.
			if tc.deleted {
				fakeClusterReader.getResource = nil
				fakeClusterReader.getErr = errors.NewNotFound(schema.GroupResource{Group: tc.gvk.Group, Resource: tc.gvk.Kind}, "fake-name")
			}

			fakeMapper := fakemapper.NewFakeRESTMapper(tc.gvk)
			statusReader := &ConfigConnectorStatusReader{
				Mapper: fakeMapper,
			}

			res, err := statusReader.ReadStatus(context.Background(), fakeClusterReader, object.UnstructuredToObjMetadata(obj))
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedStatus, res.Status)
		})
	}
}
