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
	"fmt"

	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
	"github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/cache"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EngineOption interface {
	apply(engine *cadEngine) error
}

type EngineOptionFunc func(engine *cadEngine) error

var _ EngineOption = EngineOptionFunc(nil)

func (f EngineOptionFunc) apply(engine *cadEngine) error {
	return f(engine)
}

func WithCache(cache *cache.Cache) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.cache = cache
		return nil
	})
}

func WithGRPCFunctionRuntime(address string) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		runtime, err := createGRPCFunctionRuntime(address)
		if err != nil {
			return fmt.Errorf("failed to create function runtime: %w", err)
		}
		engine.runtime = runtime
		return nil
	})
}

func WithKubeFunctionRuntime(coreClient client.Client, ns string) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		runtime, err := NewKubeFunctionRuntime(coreClient, ns)
		if err != nil {
			return fmt.Errorf("failed to create function runtime: %w", err)
		}
		engine.runtime = runtime
		return nil
	})
}

func WithFunctionRuntime(runtime fn.FunctionRuntime) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.runtime = runtime
		return nil
	})
}

func WithSimpleFunctionRuntime() EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.runtime = kpt.NewSimpleFunctionRuntime()
		return nil
	})
}

func WithRenderer(renderer fn.Renderer) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.renderer = renderer
		return nil
	})
}

func WithCredentialResolver(resolver repository.CredentialResolver) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.credentialResolver = resolver
		return nil
	})
}

func WithReferenceResolver(resolver ReferenceResolver) EngineOption {
	return EngineOptionFunc(func(engine *cadEngine) error {
		engine.referenceResolver = resolver
		return nil
	})
}

func createGRPCFunctionRuntime(address string) (kpt.FunctionRuntime, error) {
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
