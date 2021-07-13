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

package resolver

import (
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/apply/taskrunner"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
	"sigs.k8s.io/cli-utils/pkg/object"
)

func TestLiveErrorResolver(t *testing.T) {
	testCases := map[string]struct {
		err      error
		expected string
	}{
		"nested timeoutError": {
			err: &errors.Error{
				Err: &taskrunner.TimeoutError{
					Identifiers: []object.ObjMetadata{
						{
							GroupKind: schema.GroupKind{
								Group: "apps",
								Kind:  "Deployment",
							},
							Name:      "test",
							Namespace: "test-ns",
						},
					},
					Condition: taskrunner.AllCurrent,
					Timeout:   3 * time.Second,
					TimedOutResources: []taskrunner.TimedOutResource{
						{
							Identifier: object.ObjMetadata{
								GroupKind: schema.GroupKind{
									Group: "apps",
									Kind:  "Deployment",
								},
								Name:      "test",
								Namespace: "test-ns",
							},
							Status:  status.InProgressStatus,
							Message: "this is a test",
						},
					},
				},
			},
			expected: `
Error: Timeout after 3 seconds waiting for 1 out of 1 resources to reach condition AllCurrent:

Deployment/test InProgress this is a test
`,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			res, ok := (&liveErrorResolver{}).Resolve(tc.err)
			if !ok {
				t.Error("expected error to be resolved, but it wasn't")
			}
			assert.Equal(t, strings.TrimSpace(tc.expected), strings.TrimSpace(res.Message))
		})
	}
}
