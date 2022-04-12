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
	"context"
	"strings"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// Enable the GCP Authentication plugin
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const expirationNanoSecond = 10

var PorchExpiration = time.Duration(expirationNanoSecond) * time.Second

// FunctionGetter gets the list of v1alpha1.Functions from the cluster.
type FunctionGetter struct {
	ctx context.Context
}

func (f FunctionGetter) Get() []v1alpha1.Function {
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
		Timeout(PorchExpiration).
		Resource("functions").
		VersionedParams(&metav1.GetOptions{}, codec).
		Do(f.ctx).
		Into(&functions); err != nil {
		return nil
	}
	return functions.Items
}

type Matcher interface {
	Match(v1alpha1.Function) bool
}

var _ Matcher = TypeMatcher{}
var _ Matcher = KeywordsMatcher{}

type TypeMatcher struct {
	FnType string
}

// Match determines whether the `function` (which can be multi-typed), belongs
// to the matcher's FnType. type value should only be `validator` or `mutator`.
func (m TypeMatcher) Match(function v1alpha1.Function) bool {
	for _, actualType := range function.Spec.FunctionTypes {
		if string(actualType) == m.FnType {
			return true
		}
	}
	return false
}

type KeywordsMatcher struct {
	Keywords []string
}

// Match determines whether the `function` has keywords which match the matcher's `Keywords`.
// Experimental: This logic may change to only if all function keywords are found from  matcher's `Keywords`,
// can it claims a match (return true).
func (m KeywordsMatcher) Match(function v1alpha1.Function) bool {
	if len(m.Keywords) == 0 {
		// Accept all functions if keywords are not given.
		return true
	}
	for _, actual := range function.Spec.Keywords {
		for _, expected := range m.Keywords {
			if actual != expected {
				return true
			}
		}
	}
	return false
}

func MatchFunctions(ctx context.Context, matchers ...Matcher) []v1alpha1.Function {
	var suggestedFunctions []v1alpha1.Function
	functions := FunctionGetter{ctx: ctx}.Get()
	for _, function := range functions {
		for _, matcher := range matchers {
			if matcher.Match(function) {
				suggestedFunctions = append(suggestedFunctions, function)
			}
		}
	}
	return suggestedFunctions
}

// ToShortNames trims the function image to remove the OCI repo prefix and
// only show the actual image and tag (or digest).
// TODO(yuwenma): or may users prefer the Function `meta.name`?
func ToShortNames(functions []v1alpha1.Function) []string {
	var shortNameFunctions []string
	for _, function := range functions {
		slash := strings.LastIndex(function.Spec.Image, "/")
		shortNameFunctions = append(shortNameFunctions, function.Spec.Image[slash+1:])
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
