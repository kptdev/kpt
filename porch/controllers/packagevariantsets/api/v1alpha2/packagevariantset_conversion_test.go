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

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/GoogleContainerTools/kpt/porch/controllers/packagevariantsets/api/v1alpha1"
)

func TestValidateConvertTo(t *testing.T) {
	testCases := map[string]struct {
		from        PackageVariantSet
		to          v1alpha1.PackageVariantSet
		expectedErr string
	}{
		"empty": {
			from:        PackageVariantSet{},
			to:          v1alpha1.PackageVariantSet{},
			expectedErr: "",
		},
		"empty spec": {
			from: PackageVariantSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
			},
			to: v1alpha1.PackageVariantSet{
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
			var to v1alpha1.PackageVariantSet
			err := tc.from.ConvertTo(&to)
			require.NoError(t, err)
			require.Equal(t, tc.to, to)
		})
	}
}

func TestValidateConvertFrom(t *testing.T) {
	testCases := map[string]struct {
		from        v1alpha1.PackageVariantSet
		to          PackageVariantSet
		expectedErr string
	}{
		"empty": {
			from:        v1alpha1.PackageVariantSet{},
			to:          PackageVariantSet{},
			expectedErr: "",
		},
		"empty spec": {
			from: v1alpha1.PackageVariantSet{
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
			err := to.ConvertFrom(&tc.from)
			require.NoError(t, err)
			require.Equal(t, tc.to, to)
		})
	}
}
