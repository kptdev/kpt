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

package repository

import (
	"fmt"
	"testing"

	v1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestValidateWorkspaceName(t *testing.T) {
	testCases := map[string]struct {
		workspace   string
		expectedErr error
	}{
		"empty string": {
			workspace:   "",
			expectedErr: fmt.Errorf("workspaceName %q must be at least 1 and at most 63 characters long", ""),
		},

		"64 characters long": {
			workspace:   "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsdk",
			expectedErr: fmt.Errorf("workspaceName %q must be at least 1 and at most 63 characters long", "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsdk"),
		},

		"63 characters long": {
			workspace:   "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsk",
			expectedErr: nil,
		},

		"starts with -": {
			workspace:   "-hello",
			expectedErr: fmt.Errorf("workspaceName %q must start and end with an alphanumeric character", "-hello"),
		},

		"ends with -": {
			workspace:   "hello-",
			expectedErr: fmt.Errorf("workspaceName %q must start and end with an alphanumeric character", "hello-"),
		},

		"has - in the middle": {
			workspace:   "hel-lo-wor-ld",
			expectedErr: nil,
		},

		"has uppercase alphanumeric characters": {
			workspace:   "hElLo",
			expectedErr: fmt.Errorf("workspaceName %q must be comprised only of lowercase alphanumeric characters and '-'", "hElLo"),
		},

		"has other characters": {
			workspace:   "hel lo",
			expectedErr: fmt.Errorf("workspaceName %q must be comprised only of lowercase alphanumeric characters and '-'", "hel lo"),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			got := ValidateWorkspaceName(v1alpha1.WorkspaceName(tc.workspace))
			assert.Equal(t, tc.expectedErr, got)
		})
	}
}
