// Copyright 2021 Google LLC
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
	goerrors "errors"

	"github.com/GoogleContainerTools/kpt/internal/errors"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&fnExecErrorResolver{})
	AddErrorResolver(&alreadyHandledErrorResolver{})
}

// gitExecErrorResolver is an implementation of the ErrorResolver interface
// that can produce error messages for errors of the FnExecError type.
type fnExecErrorResolver struct{}

func (*fnExecErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	kioErr := errors.UnwrapKioError(err)

	var fnErr *errors.FnExecError
	if !goerrors.As(kioErr, &fnErr) {
		return ResolvedResult{}, false
	}

	return ResolvedResult{
		Message:  fnErr.String(),
		ExitCode: 1,
	}, true
}

type alreadyHandledErrorResolver struct{}

func (*alreadyHandledErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	kioErr := errors.UnwrapKioError(err)
	if goerrors.Is(kioErr, errors.ErrAlreadyHandled) {
		return ResolvedResult{ExitCode: 1}, true
	}
	return ResolvedResult{}, false
}
