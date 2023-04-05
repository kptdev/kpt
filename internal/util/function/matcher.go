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
)

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
	if m.FnType == "" {
		// type is not given, shown all functions.
		return true
	}
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
			if actual == expected {
				return true
			}
		}
	}
	return false
}

func MatchFunctions(functions []v1alpha1.Function, matchers ...Matcher) []v1alpha1.Function {
	var suggestedFunctions []v1alpha1.Function
	for _, function := range functions {
		match := true
		for _, matcher := range matchers {
			if !matcher.Match(function) {
				match = false
			}
		}
		if match {
			suggestedFunctions = append(suggestedFunctions, function)
		}
	}
	return suggestedFunctions
}

// GetNames returns the list of function names.
// - Porch function name is <PackageRepository>:<ImageName>:<Version>. e.g. kpt-functions:set-annotation:v0.1
// - Catalog v2 function name is trimed to only contain <ImageName>:<Version>, and exclude gcr.io/kpt-fn. e.g. set-annotation:v0.1
func GetNames(functions []v1alpha1.Function) []string {
	var names []string
	for _, function := range functions {
		var name string
		if function.Namespace == CatalogV2 {
			name = function.Name
		} else {
			name = fmt.Sprintf("%v:%v", function.Namespace, function.Name)
		}
		names = append(names, name)
	}
	return names
}
