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
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// Enable the GCP Authentication plugin
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const Expiration time.Duration = 10 * time.Second

// FunctionListGetter gets the list of v1alpha1.Functions from the cluster.
type FunctionListGetter struct{}

func (f FunctionListGetter) Get(ctx context.Context) []v1alpha1.Function {
	kubeflags := genericclioptions.NewConfigFlags(true)
	client, err := CreateRESTClient(kubeflags)
	if err != nil {
		return nil
	}
	scheme := runtime.NewScheme()
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return nil
	}
	codec := runtime.NewParameterCodec(scheme)
	var functions v1alpha1.FunctionList
	if err := client.Get().
		Timeout(Expiration).
		Resource("functions").
		VersionedParams(&metav1.GetOptions{}, codec).
		Do(ctx).
		Into(&functions); err != nil {
		return nil
	}
	return functions.Items
}

// FunctionGetter gets a specific v1alpha1.Functions by name.
type FunctionGetter struct{}

func (f FunctionGetter) Get(ctx context.Context, name, namespace string) (v1alpha1.Function, error) {
	kubeflags := genericclioptions.NewConfigFlags(true)
	var function v1alpha1.Function
	client, err := CreateRESTClient(kubeflags)
	if err != nil {
		return function, err
	}
	scheme := runtime.NewScheme()
	if err := v1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		return function, err
	}
	codec := runtime.NewParameterCodec(scheme)
	err = client.Get().
		Timeout(Expiration).
		Resource("functions").
		Name(name).
		Namespace(namespace).
		VersionedParams(&metav1.GetOptions{}, codec).
		Do(ctx).
		Into(&function)
	return function, err
}

func ToShortNames(functions []v1alpha1.Function) []string {
	var shortNameFunctions []string
	for _, function := range functions {
		shortNameFunctions = append(shortNameFunctions, function.Name)
	}
	return shortNameFunctions
}

func UnifyKeywords(functions []v1alpha1.Function) []string {
	var keywords []string
	keywordsMap := map[string]bool{}
	for _, function := range functions {
		for _, kw := range function.Spec.Keywords {
			if _, ok := keywordsMap[kw]; !ok {
				keywordsMap[kw] = true
				keywords = append(keywords, kw)
			}
		}
	}
	return keywords
}
