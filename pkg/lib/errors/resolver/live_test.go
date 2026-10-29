// Copyright 2021,2026 The kpt Authors
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

	"github.com/kptdev/kpt/pkg/lib/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

func TestLiveErrorResolver(t *testing.T) {
	testCases := map[string]struct {
		err      error
		expected string
	}{
		"nested unknown type": {
			err: &errors.Error{
				Err: &manifestreader.UnknownTypesError{
					GroupVersionKinds: []schema.GroupVersionKind{
						{
							Group:   "apps",
							Version: "v1",
							Kind:    "Deployment",
						},
					},
				},
			},
			expected: `
Error: The following resource types are not available in the cluster:

Resource types:
apps/v1, Kind=Deployment

Please remove or update these resources, or install the required CRD(s) if they are custom resources.
`,
		},
		"multiple nested unknown types": {
			err: &errors.Error{
				Err: &manifestreader.UnknownTypesError{
					GroupVersionKinds: []schema.GroupVersionKind{
						{
							Group:   "policy",
							Version: "v1beta1",
							Kind:    "PodSecurityPolicy",
						},
						{
							Group:   "example.com",
							Version: "v1",
							Kind:    "Widget",
						},
					},
				},
			},
			expected: `
Error: The following resource types are not available in the cluster:

Resource types:
policy/v1beta1, Kind=PodSecurityPolicy
example.com/v1, Kind=Widget

Please remove or update these resources, or install the required CRD(s) if they are custom resources.
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
