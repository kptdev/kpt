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

package engine

import (
	"context"
	"errors"
	"io"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
)

type delegatingRuntime struct {
	builtin *builtinRuntime
	grpc    *grpcRuntime
}

func newDelegatingFunctionRuntime(addr string) (*delegatingRuntime, error) {
	builtin := newBuiltinRuntime()
	gprc, err := newGRPCFunctionRuntime(addr)
	if err != nil {
		return nil, err
	}
	return &delegatingRuntime{
		builtin: builtin,
		grpc:    gprc,
	}, nil
}

var _ kpt.FunctionRuntime = &delegatingRuntime{}

func (dr *delegatingRuntime) GetRunner(ctx context.Context, fn *v1.Function) (fn.FunctionRunner, error) {
	br, err := dr.builtin.GetRunner(ctx, fn)
	var unsupportedErr *UnsupportedFunctionError
	if err == nil || !errors.As(err, &unsupportedErr) {
		return &delegatingRunner{
			runner: br,
		}, err
	}
	// If the error is of UnsupportedFunctionError type, we try the gRPC runner.
	gr, err := dr.grpc.GetRunner(ctx, fn)
	return &delegatingRunner{
		runner: gr,
	}, err
}

func (dr *delegatingRuntime) Close() error {
	if err := dr.builtin.Close(); err != nil {
		return err
	}
	return dr.grpc.Close()
}

type delegatingRunner struct {
	runner fn.FunctionRunner
}

var _ fn.FunctionRunner = &delegatingRunner{}

func (dr *delegatingRunner) Run(r io.Reader, w io.Writer) error {
	return dr.runner.Run(r, w)
}
