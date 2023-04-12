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
	"fmt"
	"io"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
)

// FunctionRunner knows how to run a function.
type FunctionRunner interface {
	// Run method accepts resourceList in wireformat and returns resourceList in wire format.
	Run(r io.Reader, w io.Writer) error
}

// FunctionRuntime provides a way to obtain a function runner to be used for a given function configuration.
// If the function is not found, this should return an error that includes a NotFoundError in the chain.
type FunctionRuntime interface {
	GetRunner(ctx context.Context, fn *v1.Function) (FunctionRunner, error)
}

type NotFoundError struct {
	Function v1.Function
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("function %q not found", e.Function.Image)
}
