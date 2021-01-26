// Copyright 2020 Google LLC
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

package search

import (
	"strings"
)

// pathMatch checks if the traversed yaml path matches with the user input path
// checks if user input path is valid
func (sr *SearchReplace) pathMatch(yamlPath string) bool {
	if sr.ByPath == "" {
		return false
	}

	patternElems := strings.Split(sr.ByPath, PathDelimiter)
	yamlPathElems := strings.Split(strings.TrimPrefix(yamlPath, PathDelimiter), PathDelimiter)
	return backTrackMatch(yamlPathElems, patternElems)
}

// backTrackMatch matches the traversed yamlPathElems with input(from by-path) patternElems
// * matches any element, ** matches 0 or more elements, array elements are split and matched
// refer to pathparser_test.go
func backTrackMatch(yamlPathElems, patternElems []string) bool {
	// this is a dynamic programming problem
	// aim is to check if path array matches pattern array as per above rules
	yamlPathElemsLen, patternElemsLen := len(yamlPathElems), len(patternElems)

	// initialize a 2d boolean matrix to memorize results
	// dp[i][j] stores the result, if yamlPath subarray of length i matches
	// pattern subarray of length j
	dp := make([][]bool, yamlPathElemsLen+1)
	for i := range dp {
		dp[i] = make([]bool, patternElemsLen+1)
	}
	dp[0][0] = true

	// edge case 1: when pattern is empty, yamlPath of length grater than 0 doesn't match
	for i := 1; i < yamlPathElemsLen+1; i++ {
		dp[i][0] = false
	}

	// edge case 2: if yamlPath is empty, carry forward the previous result if the pattern element
	// is `**` as it matches 0 or more elements.
	for j := 1; j < patternElemsLen+1; j++ {
		if patternElems[j-1] == "**" {
			dp[0][j] = dp[0][j-1]
		}
	}

	// fill rest of the matrix
	for i := 1; i < yamlPathElemsLen+1; i++ {
		for j := 1; j < patternElemsLen+1; j++ {
			if patternElems[j-1] == "**" {
				// `**` matches multiple elements, so carry forward the result from immediate
				// neighbors, dp[i-1][j] match empty, dp[i][j-1] match multiple elements
				dp[i][j] = dp[i][j-1] || dp[i-1][j]
			} else if patternElems[j-1] == "*" || elementMatch(yamlPathElems[i-1], patternElems[j-1]) {
				// if there is element match or `*` then get the result from previous diagonal element
				dp[i][j] = dp[i-1][j-1]
			}
		}
	}

	/*Example matrix for yamlPath = [a,a,b,c,e,b] and pattern [a,*,b,**,b]
		  a	a	b	c	e	b
		a	T	F	F	F	F	F
	  * F	T	F	F	F	F
		b	F	F	T	F	F	F
	 ** F	F	T	T	T	T
		b	F	F	F	F	F	T
	*/

	return dp[yamlPathElemsLen][patternElemsLen]
}

// elementMatch matches single element with pattern for single element
func elementMatch(elem, pattern string) bool {
	// scalar field case `metadata` matches `metadata`
	if elem == pattern {
		return true
	}
	// array element e.g. a[*], *[*] and *[b] matches a[b]
	if strings.Contains(elem, "[") {
		elemParts := strings.Split(elem, "[")
		patternParts := strings.Split(pattern, "[")
		if patternParts[0] != "*" && elemParts[0] != patternParts[0] {
			return false
		}
		return patternParts[1] == "*]" || elemParts[1] == patternParts[1]
	}
	return false
}

// isAbsPath checks if input path is absolute and not a path expression
// only supported path format is e.g. foo.bar.baz
func isAbsPath(path string) bool {
	pathElem := strings.Split(path, PathDelimiter)
	if len(pathElem) == 0 {
		return false
	}
	for _, elem := range pathElem {
		// more checks can be added in future
		if elem == "" || strings.Contains(elem, "*") {
			return false
		}
	}
	return true
}
