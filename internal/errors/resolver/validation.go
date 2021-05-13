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
	"errors"

	kptfile "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1alpha2"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&kptfileValidateErrorResolver{})
}

// kptfileValidateErrorResolver is an implementation of the ErrorResolver interface
// to resolve Kptfile validation error.
type kptfileValidateErrorResolver struct{}

func (*kptfileValidateErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var validateError *kptfile.ValidateError
	if !errors.As(err, &validateError) {
		return ResolvedResult{}, false
	}
	return ResolvedResult{
		Message:  validateError.Error(),
		ExitCode: 1,
	}, true
}
