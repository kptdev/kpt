// Copyright 2023 The kpt Authors
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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha2 "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha2"
)

func TestValidateConvertV1alpha2ToV1alpha1(t *testing.T) {
	testCases := map[string]struct {
		from        v1alpha2.PackageVariantSet
		to          PackageVariantSet
		expectedErr string
	}{
		"empty": {
			from:        v1alpha2.PackageVariantSet{},
			to:          PackageVariantSet{},
			expectedErr: "",
		},
		"empty spec": {
			from: v1alpha2.PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			to: PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			expectedErr: "",
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var to PackageVariantSet
			err := ConvertV1alpha2ToV1alpha1(&tc.from, &to)
			require.NoError(t, err)
			require.Equal(t, tc.to, to)
		})
	}
}

func TestValidateConvertV1alpha1ToV1alpha2(t *testing.T) {
	testCases := map[string]struct {
		from        PackageVariantSet
		to          v1alpha2.PackageVariantSet
		expectedErr string
	}{
		"empty": {
			from:        PackageVariantSet{},
			to:          v1alpha2.PackageVariantSet{},
			expectedErr: "",
		},
		"empty spec": {
			from: PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			to: v1alpha2.PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			expectedErr: "",
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			var to v1alpha2.PackageVariantSet
			err := ConvertV1alpha1ToV1alpha2(&tc.from, &to)
			require.NoError(t, err)
			require.Equal(t, tc.to, to)
		})
	}
}
