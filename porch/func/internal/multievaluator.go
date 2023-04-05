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

package internal

import (
	"context"
	"errors"

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Evaluator interface {
	EvaluateFunction(context.Context, *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error)
}

type multiEvaluator struct {
	pb.UnimplementedFunctionEvaluatorServer

	evaluators []Evaluator
}

func NewMultiEvaluator(evaluators ...Evaluator) pb.FunctionEvaluatorServer {
	return &multiEvaluator{
		evaluators: evaluators,
	}
}

func (me *multiEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	var err error
	var notFoundErr *fn.NotFoundError
	var resp *pb.EvaluateFunctionResponse
	for _, eval := range me.evaluators {
		resp, err = eval.EvaluateFunction(ctx, req)
		if err == nil {
			return resp, nil
		} else if !errors.As(err, &notFoundErr) {
			return nil, err
		}
	}
	return nil, status.Error(codes.NotFound, err.Error())
}
