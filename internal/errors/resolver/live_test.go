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

package resolver

import (
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/cli-utils/pkg/manifestreader"
)

func TestLiveErrorResolver(t *testing.T) {
	testCases := map[string]struct {
		err      error
		expected string
	}{
		"nested timeoutError": {
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
Error: 1 resource types could not be found in the cluster or as CRDs among the applied resources.

Resource types:
apps/v1, Kind=Deployment
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
