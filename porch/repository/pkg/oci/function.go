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

package oci

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ociFunction struct {
	ref     name.Reference
	tag     name.Reference
	name    string
	version string
	created time.Time

	parent *ociRepository
}

var _ repository.Function = &ociFunction{}

func (f *ociFunction) Name() string {
	return fmt.Sprintf("%s:%s:%s", f.parent.name, f.name, f.version)
}

func (f *ociFunction) GetFunction() (*v1alpha1.Function, error) {
	return &v1alpha1.Function{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.Name(),
			Namespace: f.parent.namespace,
			UID:       "",
			CreationTimestamp: metav1.Time{
				Time: f.created,
			},
		},
		Spec: v1alpha1.FunctionSpec{
			Image: f.tag.Name(),
			RepositoryRef: v1alpha1.RepositoryRef{
				Name: f.parent.name,
			},
			FunctionType:       "TODO",
			Description:        "TODO",
			DocumentationUrl:   "TODO",
			InputTypes:         []metav1.TypeMeta{},
			OutputTypes:        []metav1.TypeMeta{},
			FunctionConfigType: metav1.TypeMeta{},
		},
		Status: v1alpha1.FunctionStatus{},
	}, nil

}

// RepositoryStr is the repository part of the resource name,
// typically slash-separated. Last segment is the base function name.
func parseFunctionName(repositoryStr string) string {
	slash := strings.LastIndex(repositoryStr, "/")
	return repositoryStr[slash+1:]
}
