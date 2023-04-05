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

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestResolveError_DefaultExitCode(t *testing.T) {
	org := errorResolvers
	AddErrorResolver(&TestErrorResolver{})
	defer func() {
		errorResolvers = org
	}()

	rr, ok := ResolveError(&TestError{})
	assert.True(t, ok)
	assert.Equal(t, 1, rr.ExitCode)
}

type TestErrorResolver struct{}

func (t *TestErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var testError *TestError
	if errors.As(err, &testError) {
		return ResolvedResult{}, true
	}
	return ResolvedResult{}, false
}

type TestError struct{}

func (e *TestError) Error() string {
	return "this is a test"
}
