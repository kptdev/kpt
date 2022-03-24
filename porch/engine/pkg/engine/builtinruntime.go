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
	"fmt"
	"io"
	"io/ioutil"

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
			"gcr.io/kpt-fn/apply-replacements@sha256:c7a1b2d96e7f02e8aca6b1d15c15542c73f5a9f072c7fc6a3569160f8662e056": fnsdk.ResourceListProcessorFunc(replacements.ApplyReplacements),
			"gcr.io/kpt-fn/set-namespace:unstable":                                                                     fnsdk.ResourceListProcessorFunc(transformer.SetNamespace),
			"gcr.io/kpt-fn/set-namespace@sha256:36bf1a791436a75c0108c071bdb34c2cfe97329d548ac47329119dad87d6c6e2":      fnsdk.ResourceListProcessorFunc(transformer.SetNamespace),
		},
	}
}

var _ kpt.FunctionRuntime = &builtinRuntime{}

func (br *builtinRuntime) GetRunner(ctx context.Context, fn *v1.Function) (fn.FunctionRunner, error) {
	processor, found := br.fnMapping[fn.Image]
	if !found {
		return nil, &UnsupportedFunctionError{Image: fn.Image}
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
	in, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read function runner input: %w", err)
	}

	rl, err := fnsdk.ParseResourceList(in)
	if err != nil {
		return fmt.Errorf("failed to parse the resourceList: %w", err)
	}
	if err = br.processor.Process(rl); err != nil {
		return fmt.Errorf("func eval failed: %w", err)
	}
	y, err := rl.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal resourceList as yaml: %w", err)
	}
	if _, err = w.Write(y); err != nil {
		return fmt.Errorf("failed to write function runner output: %w", err)
	}
	return nil
}
