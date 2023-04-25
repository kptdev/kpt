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

package fn

import (
	"context"
	"errors"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

// MultiRuntime is a compound FunctionRuntime that will use the first available runner.
type MultiRuntime struct {
	runtimes []FunctionRuntime
}

// NewMultiRuntime builds a MultiRuntime.
func NewMultiRuntime(runtimes []FunctionRuntime) *MultiRuntime {
	return &MultiRuntime{
		runtimes: runtimes,
	}
}

var _ FunctionRuntime = &MultiRuntime{}

// GetRunner implements FunctionRuntime
func (r *MultiRuntime) GetRunner(ctx context.Context, fn *v1.Function) (FunctionRunner, error) {
	for _, runtime := range r.runtimes {
		runner, err := runtime.GetRunner(ctx, fn)
		if err != nil {
			var notFoundError *NotFoundError
			if !errors.As(err, &notFoundError) {
				return nil, err
			}
		} else {
			return runner, nil
		}
	}

	return nil, &NotFoundError{Function: *fn}
}

// Add adds the provided runtime to the end of the MultiRuntime list.
func (r *MultiRuntime) Add(runtime FunctionRuntime) {
	r.runtimes = append(r.runtimes, runtime)
}
