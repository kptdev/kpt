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

package function

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CatalogV2 is the an invalid namespace used to build catalog function into the porch v1alpha1.function struct.
// This namespace distinguishes catalog function from porch functions. This will be trimmed before showing to users.
const CatalogV2 = "__catalog_v2"

// CatalogFunction converts catalog function into the porch v1alpha1.function struct.
func CatalogFunction(name string, keywords []string, fnTypes []v1alpha1.FunctionType) v1alpha1.Function {
	return v1alpha1.Function{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: CatalogV2,
		},
		Spec: v1alpha1.FunctionSpec{
			Image:         fmt.Sprintf("gcr.io/kpt-fn/%s", name),
			FunctionTypes: fnTypes,
			Keywords:      keywords,
		},
	}
}
