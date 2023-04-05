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

package oci

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/pkg/repository"
	"github.com/google/go-containerregistry/pkg/name"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ociImagePrefix      = "dev.kpt.fn.meta."
	FunctionTypesKey    = ociImagePrefix + "types"
	DescriptionKey      = ociImagePrefix + "description"
	DocumentationURLKey = ociImagePrefix + "documentationurl"
	keywordsKey         = ociImagePrefix + "keywords"

	fnConfigMetaPrefix = ociImagePrefix + "fnconfig."
	// experimental: this field is very likely to be changed in the future.
	ConfigMapFnKey = fnConfigMetaPrefix + "configmap.requiredfields"
)

func AnnotationToSlice(annotation string) []string {
	vals := strings.Split(annotation, ",")
	var result []string
	for _, val := range vals {
		result = append(result, strings.TrimSpace(val))
	}
	return result
}

type functionMeta struct {
	FunctionTypes    []string
	Description      string
	DocumentationUrl string
	Keywords         []string
	// experimental: this field is very likely to be changed in the future.
	FunctionConfigs []functionConfig
}

// experimental: this struct is very likely to be changed in the future.
type functionConfig struct {
	metav1.TypeMeta
	RequiredFields []string
}

type ociFunction struct {
	ref     name.Reference
	tag     name.Reference
	name    string
	version string
	created time.Time
	meta    *functionMeta
	parent  *ociRepository
}

var _ repository.Function = &ociFunction{}

// LINT.IfChange(Name)
func (f *ociFunction) Name() string {
	return fmt.Sprintf("%s:%s:%s", f.parent.name, f.name, f.version)
	// LINT.ThenChange(internal/fnruntime/container.go AddDefaultImagePathPrefix)
}

func (f *ociFunction) GetFunction() (*v1alpha1.Function, error) {
	var functionTypes []v1alpha1.FunctionType
	for _, fnType := range f.meta.FunctionTypes {
		switch {
		case fnType == string(v1alpha1.FunctionTypeMutator):
			functionTypes = append(functionTypes, v1alpha1.FunctionTypeMutator)
		case fnType == string(v1alpha1.FunctionTypeValidator):
			functionTypes = append(functionTypes, v1alpha1.FunctionTypeValidator)
		default:
			// unrecognized custom FunctionType
		}
	}
	var fnConfigs []v1alpha1.FunctionConfig
	for _, metaFnConfig := range f.meta.FunctionConfigs {
		fnConfigs = append(fnConfigs, v1alpha1.FunctionConfig{
			TypeMeta:       metaFnConfig.TypeMeta,
			RequiredFields: metaFnConfig.RequiredFields,
		})
	}
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
			FunctionTypes:    functionTypes,
			Description:      f.meta.Description,
			DocumentationUrl: f.meta.DocumentationUrl,
			Keywords:         f.meta.Keywords,
			FunctionConfigs:  fnConfigs,
		},
		Status: v1alpha1.FunctionStatus{},
	}, nil
}

func (f *ociFunction) GetCRD() (*configapi.Function, error) {
	var functionTypes []configapi.FunctionType
	for _, fnType := range f.meta.FunctionTypes {
		switch {
		case fnType == string(v1alpha1.FunctionTypeMutator):
			functionTypes = append(functionTypes, configapi.FunctionTypeMutator)
		case fnType == string(v1alpha1.FunctionTypeValidator):
			functionTypes = append(functionTypes, configapi.FunctionTypeValidator)
		default:
			// unrecognized custom FunctionType
		}
	}
	var fnConfigs []configapi.FunctionConfig
	for _, metaFnConfig := range f.meta.FunctionConfigs {
		fnConfigs = append(fnConfigs, configapi.FunctionConfig{
			TypeMeta:       metaFnConfig.TypeMeta,
			RequiredFields: metaFnConfig.RequiredFields,
		})
	}

	name := fmt.Sprintf("%s.%s.%s", f.parent.name, f.name, f.version)

	return &configapi.Function{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: f.parent.namespace,
		},
		Spec: configapi.FunctionSpec{
			Image: f.tag.Name(),
			RepositoryRef: configapi.RepositoryRef{
				Name: f.parent.name,
			},
			FunctionTypes:    functionTypes,
			Description:      f.meta.Description,
			DocumentationUrl: f.meta.DocumentationUrl,
			Keywords:         f.meta.Keywords,
			FunctionConfigs:  fnConfigs,
		},
	}, nil
}

// RepositoryStr is the repository part of the resource name,
// typically slash-separated. Last segment is the base function name.
func parseFunctionName(repositoryStr string) string {
	slash := strings.LastIndex(repositoryStr, "/")
	return repositoryStr[slash+1:]
}
