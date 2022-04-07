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
	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/starlark/starlark"
	fnsdk "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/kpt"
)

var (
	applyReplacementsImageAliases = []string{
		"gcr.io/kpt-fn/apply-replacements:v0.1.0",
		"gcr.io/kpt-fn/apply-replacements:v0.1",
		"gcr.io/kpt-fn/apply-replacements@sha256:40d00367d46c04088d68ebd05649e1bff6ea43be3a2d3f4d257eef18c4d70f8c",
	}
	setNamespaceImageAliases = []string{
		"gcr.io/kpt-fn/set-namespace:v0.3.1",
		"gcr.io/kpt-fn/set-namespace:v0.3",
		"gcr.io/kpt-fn/set-namespace@sha256:ea61ed9ea562cefaa2c2f256e8011352221cc45844aa7c9a61ba6b781b5dba47",
	}
	starlarkImageAliases = []string{
		"gcr.io/kpt-fn/starlark:v0.4.0",
		"gcr.io/kpt-fn/starlark:v0.4",
		"gcr.io/kpt-fn/starlark@sha256:4f98e9298eb1d605ec22515771e0c495dad75606058edd926449825dc59dcd1b",
	}
)

type builtinRuntime struct {
	fnMapping map[string]fnsdk.ResourceListProcessor
}

func newBuiltinRuntime() *builtinRuntime {
	fnMap := map[string]fnsdk.ResourceListProcessor{}
	for _, img := range applyReplacementsImageAliases {
		fnMap[img] = fnsdk.ResourceListProcessorFunc(replacements.ApplyReplacements)
	}
	for _, img := range setNamespaceImageAliases {
		fnMap[img] = fnsdk.ResourceListProcessorFunc(transformer.SetNamespace)
	}
	for _, img := range starlarkImageAliases {
		fnMap[img] = fnsdk.ResourceListProcessorFunc(starlark.Process)
	}
	return &builtinRuntime{
		fnMapping: fnMap,
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
