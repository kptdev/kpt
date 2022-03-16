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
	"context"

	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/apply-replacements/replacements"
	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/starlark/starlark"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"k8s.io/klog/v2"
)

type builtinEvaluator struct {
	pb.UnimplementedFunctionEvaluatorServer

	fnMapping map[string]fn.ResourceListProcessor
}

func newBuiltInEvaluator() (*builtinEvaluator, error) {
	return &builtinEvaluator{
		fnMapping: map[string]fn.ResourceListProcessor{
			"gcr.io/kpt-fn/starlark:v0.3.0": &starlark.Processor{},
			"gcr.io/kpt-fn/starlark@sha256:c347e28606fa1a608e8e02e03541a5a46e4a0152005df4a11e44f6c4ab1edd9a":           &starlark.Processor{},
			"gcr.io/kpt-fn/apply-replacements:unstable":                                                                fn.ResourceListProcessorFunc(replacements.ApplyReplacements),
			"gcr.io/kpt-fn/apply-replacements@sha256:c7a1b2d96e7f02e8aca6b1d15c15542c73f5a9f072c7fc6a3569160f8662e056": fn.ResourceListProcessorFunc(replacements.ApplyReplacements),
		},
	}, nil
}

func (e *builtinEvaluator) EvaluateFunction(ctx context.Context, req *pb.EvaluateFunctionRequest) (*pb.EvaluateFunctionResponse, error) {
	processor, found := e.fnMapping[req.Image]
	if !found {
		return nil, &UnsupportedFunctionError{Image: req.Image}
	}

	rl, err := fn.ParseResourceList(req.ResourceList)
	if err != nil {
		return nil, err
	}
	if err = processor.Process(rl); err != nil {
		return nil, err
	}
	y, err := rl.ToYAML()
	if err != nil {
		return nil, err
	}
	klog.Infof("Evaluated %q: stdout %d bytes", req.Image, len(y))
	return &pb.EvaluateFunctionResponse{
		ResourceList: y,
	}, nil
}
