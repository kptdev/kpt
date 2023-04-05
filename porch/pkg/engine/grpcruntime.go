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

package engine

import (
	"context"
	"fmt"
	"io"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/pkg/kpt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
)

type grpcRuntime struct {
	cc     *grpc.ClientConn
	client evaluator.FunctionEvaluatorClient
}

func newGRPCFunctionRuntime(address string) (*grpcRuntime, error) {
	if address == "" {
		return nil, fmt.Errorf("address is required to instantiate gRPC function runtime")
	}

	klog.Infof("Dialing grpc function runner %q", address)

	cc, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc function evaluator: %w", err)
	}

	return &grpcRuntime{
		cc:     cc,
		client: evaluator.NewFunctionEvaluatorClient(cc),
	}, err
}

var _ kpt.FunctionRuntime = &grpcRuntime{}

func (gr *grpcRuntime) GetRunner(ctx context.Context, fn *v1.Function) (fn.FunctionRunner, error) {
	// TODO: Check if the function is actually available?
	return &grpcRunner{
		ctx:    ctx,
		client: gr.client,
		image:  fn.Image,
	}, nil
}

func (gr *grpcRuntime) Close() error {
	var err error
	if gr.cc != nil {
		if err = gr.cc.Close(); err != nil {
			klog.Warningf("Failed to close grpc client connection: %v", err)
		}
		gr.cc = nil
	}
	return err
}

type grpcRunner struct {
	ctx    context.Context
	client evaluator.FunctionEvaluatorClient
	image  string
}

var _ fn.FunctionRunner = &grpcRunner{}

func (gr *grpcRunner) Run(r io.Reader, w io.Writer) error {
	in, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read function runner input: %w", err)
	}

	res, err := gr.client.EvaluateFunction(gr.ctx, &evaluator.EvaluateFunctionRequest{
		ResourceList: in,
		Image:        gr.image,
	})
	if err != nil {
		return fmt.Errorf("func eval %q failed: %w", gr.image, err)
	}
	if _, err := w.Write(res.ResourceList); err != nil {
		return fmt.Errorf("failed to write function runner output: %w", err)
	}
	return nil
}
