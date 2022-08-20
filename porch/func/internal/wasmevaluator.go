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

package internal

import (
	"bytes"
	"context"

	"github.com/GoogleContainerTools/kpt/internal/fnruntime"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
)

type wasmEvaluator struct {
	cacheDir string
	// TODO: add something here if needed
}

var _ Evaluator = &wasmEvaluator{}

func NewWasmEvaluator(cacheDir string) (Evaluator, error) {
	return &wasmEvaluator{cacheDir: cacheDir}, nil
}

func (e *wasmEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	wfn, err := fnruntime.NewWasmFn(fnruntime.NewOciLoader(e.cacheDir, req.Image))
	if err != nil {
		// TODO
	}

	var buf bytes.Buffer
	if err = wfn.Run(bytes.NewReader(req.ResourceList), &buf); err != nil {
		// TODO
	}

	return &pb.EvaluateFunctionResponse{
		ResourceList: buf.Bytes(),
	}, nil
}
