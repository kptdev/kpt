// Copyright 2022 Google LLC
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
	"io"

	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type RunnerOptions struct {
	// ResultList stores the result of the function evaluation
	ResultList *fnresult.ResultList
}

type FunctionRunner interface {
	NewRunner(ctx context.Context, fn *v1.Function, opts RunnerOptions) (kio.Filter, error)
}

// FunctionExecutor knows how to run a function.
type FunctionExecutor interface {
	// Run method accepts resourceList in wireformat and returns resourceList in wire format.
	Run(r io.Reader, w io.Writer) error
}

// FunctionPicker returns a function executor to be used for a given function configuration.
type FunctionPicker func(fn *v1.Function) FunctionExecutor
