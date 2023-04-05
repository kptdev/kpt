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

package porch

import (
	"context"
	"fmt"
	"testing"

	api "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
)

func testValidateUpdate(t *testing.T, s SimpleRESTUpdateStrategy, old, new api.PackageRevisionLifecycle, valid bool) {
	ctx := context.Background()
	t.Run(fmt.Sprintf("%s-%s", old, new), func(t *testing.T) {
		oldRev := &api.PackageRevision{
			Spec: api.PackageRevisionSpec{
				Lifecycle: old,
			},
		}
		newRev := &api.PackageRevision{
			Spec: api.PackageRevisionSpec{
				Lifecycle: new,
			},
		}

		allErrs := s.ValidateUpdate(ctx, newRev, oldRev)

		if valid {
			if len(allErrs) > 0 {
				t.Errorf("Update %s -> %s failed unexpectedly: %v", old, new, allErrs.ToAggregate().Error())
			}
		} else {
			if len(allErrs) == 0 {
				t.Errorf("Update %s -> %s should fail but didn't", old, new)
			}
		}
	})
}
