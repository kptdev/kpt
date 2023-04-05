// Copyright 2021 The kpt Authors
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

package resolver

// errorResolvers is the list of known resolvers for kpt errors.
var errorResolvers []ErrorResolver

// AddErrorResolver adds the provided error resolver to the list of resolvers
// which will be used to resolve errors.
func AddErrorResolver(er ErrorResolver) {
	errorResolvers = append(errorResolvers, er)
}

// ResolveError attempts to resolve the provided error into a descriptive
// string which will be displayed to the user. If the last return value is false,
// the error could not be resolved.
func ResolveError(err error) (ResolvedResult, bool) {
	for _, resolver := range errorResolvers {
		rr, found := resolver.Resolve(err)
		// If the exit code hasn't been set, we default it to 1. We should
		// never return exit code 0 for errors.
		if rr.ExitCode == 0 {
			rr.ExitCode = 1
		}
		if found {
			return rr, true
		}
	}
	return ResolvedResult{}, false
}

type ResolvedResult struct {
	Message  string
	ExitCode int
}

// ErrorResolver is an interface that allows kpt to resolve an error into
// an error message suitable for the end user.
type ErrorResolver interface {
	Resolve(err error) (ResolvedResult, bool)
}
