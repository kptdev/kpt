// Copyright 2022 Google LLC
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

package porch

import (
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
)

func TestApprovalUpdateStrategy(t *testing.T) {
	s := packageRevisionApprovalStrategy{}

	type testCase struct {
		old     api.PackageRevisionLifecycle
		valid   []api.PackageRevisionLifecycle
		invalid []api.PackageRevisionLifecycle
	}

	for _, tc := range []testCase{
		{
			old:     "",
			valid:   []api.PackageRevisionLifecycle{},
			invalid: []api.PackageRevisionLifecycle{"", "Wrong", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecycleFinal},
		},
		{
			old:     "Wrong",
			valid:   []api.PackageRevisionLifecycle{},
			invalid: []api.PackageRevisionLifecycle{"", "Wrong", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecycleFinal},
		},
		{
			old:     api.PackageRevisionLifecycleDraft,
			valid:   []api.PackageRevisionLifecycle{},
			invalid: []api.PackageRevisionLifecycle{"", "Wrong", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecycleFinal},
		},
		{
			old:     api.PackageRevisionLifecycleFinal,
			valid:   []api.PackageRevisionLifecycle{},
			invalid: []api.PackageRevisionLifecycle{"", "Wrong", api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleProposed, api.PackageRevisionLifecycleFinal},
		},
		{
			old:     api.PackageRevisionLifecycleProposed,
			valid:   []api.PackageRevisionLifecycle{api.PackageRevisionLifecycleDraft, api.PackageRevisionLifecycleFinal},
			invalid: []api.PackageRevisionLifecycle{"", "Wrong", api.PackageRevisionLifecycleProposed},
		},
	} {
		for _, new := range tc.valid {
			testValidateUpdate(t, s, tc.old, new, true)
		}
		for _, new := range tc.invalid {
			testValidateUpdate(t, s, tc.old, new, false)
		}
	}
}
