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
	"errors"

	"github.com/kptdev/kpt/internal/fnruntime"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&containerImageErrorResolver{})
}

// containerImageErrorResolver is an implementation of the ErrorResolver interface
// to resolve container image errors.
type containerImageErrorResolver struct{}

func (*containerImageErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var containerImageError *fnruntime.ContainerImageError
	if !errors.As(err, &containerImageError) {
		return ResolvedResult{}, false
	}
	return ResolvedResult{
		Message: containerImageError.Error(),
	}, true
}
