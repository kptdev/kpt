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
	"io"

	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/apply-replacements/replacements"
	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/set-namespace/transformer"
	fnsdk "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
)

type builtinRuntime struct {
	fnMapping map[string]fnsdk.ResourceListProcessor
}

func newBuiltinRuntime() *builtinRuntime {
	return &builtinRuntime{
		fnMapping: map[string]fnsdk.ResourceListProcessor{
			"gcr.io/kpt-fn/apply-replacements:unstable":                                                                fnsdk.ResourceListProcessorFunc(replacements.ApplyReplacements),
			"gcr.io/kpt-fn/apply-replacements@sha256:fb1f749b13dc3498d411d4b3b6eda58d2599e57206c9c1d9c2b1736f48cd6ab4": fnsdk.ResourceListProcessorFunc(replacements.ApplyReplacements),
			"gcr.io/kpt-fn/set-namespace:unstable":                                                                     fnsdk.ResourceListProcessorFunc(transformer.SetNamespace),
			"gcr.io/kpt-fn/set-namespace@sha256:a18dcb30b5dda1a16d22586dae17a91cb2f53da1abfaa353eb4de9d0a2c4538f":      fnsdk.ResourceListProcessorFunc(transformer.SetNamespace),
		},
	}
}

var _ kpt.FunctionRuntime = &builtinRuntime{}

func (br *builtinRuntime) GetRunner(ctx context.Context, funct *v1.Function) (fn.FunctionRunner, error) {
	processor, found := br.fnMapping[funct.Image]
	if !found {
		return nil, &fn.NotFoundError{Function: *funct}
	}

	return &builtinRunner{
		ctx:       ctx,
		processor: processor,
	}, nil
}

func (br *builtinRuntime) Close() error {
	return nil
}

type builtinRunner struct {
	ctx       context.Context
	processor fnsdk.ResourceListProcessor
}

var _ fn.FunctionRunner = &builtinRunner{}

func (br *builtinRunner) Run(r io.Reader, w io.Writer) error {
	return fnsdk.Execute(br.processor, r, w)
}
