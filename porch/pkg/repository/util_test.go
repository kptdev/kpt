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

package repository

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDescription(t *testing.T) {
	testCases := map[string]struct {
		description string
		expectedErr error
	}{
		"empty string": {
			description: "",
			expectedErr: fmt.Errorf("description %q must be at least 1 and at most 63 characters long", ""),
		},

		"64 characters long": {
			description: "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsdk",
			expectedErr: fmt.Errorf("description %q must be at least 1 and at most 63 characters long", "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsdk"),
		},

		"63 characters long": {
			description: "abcedfhglaasdkfuaweoihfjghldhsgufhgaakjsdhaflkasdjflksadjfsalsk",
			expectedErr: nil,
		},

		"starts with -": {
			description: "-hello",
			expectedErr: fmt.Errorf("description %q must start and end with an alphanumeric character", "-hello"),
		},

		"ends with -": {
			description: "hello-",
			expectedErr: fmt.Errorf("description %q must start and end with an alphanumeric character", "hello-"),
		},

		"has - in the middle": {
			description: "hel-lo-wor-ld",
			expectedErr: nil,
		},

		"has other characters": {
			description: "hel lo",
			expectedErr: fmt.Errorf("description %q must be comprised only of alphanumeric characters and '-'", "hel lo"),
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			got := ValidateDescription(tc.description)
			assert.Equal(t, tc.expectedErr, got)
		})
	}
}
