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

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
)

//nolint:gochecknoinits
func init() {
	AddErrorResolver(&dockerImageErrorResolver{})
}

// dockerImageErrorResolver is an implementation of the ErrorResolver interface
// to resolve docker image errors.
type dockerImageErrorResolver struct{}

func (*dockerImageErrorResolver) Resolve(err error) (ResolvedResult, bool) {
	var dockerImageError *fnruntime.DockerImageError
	if !errors.As(err, &dockerImageError) {
		return ResolvedResult{}, false
	}
	return ResolvedResult{
		Message:  dockerImageError.Error(),
		ExitCode: 1,
	}, true
}
