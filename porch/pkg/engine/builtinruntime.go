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
	"io"

	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/apply-replacements/replacements"
	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/set-namespace/transformer"
	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/starlark/starlark"
	fnsdk "github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/pkg/kpt"
)

// When updating the version for the builtin functions, please also update the image version
// in test TestBuiltinFunctionEvaluator in porch/test/e2e/e2e_test.go, if the versions mismatch
// the e2e test will fail in local deployment mode.
var (
	applyReplacementsImageAliases = []string{
		"gcr.io/kpt-fn/apply-replacements:v0.1.1",
		"gcr.io/kpt-fn/apply-replacements:v0.1",
		"gcr.io/kpt-fn/apply-replacements@sha256:85913d4ec8db62053eb060ff1b7e26d13ff8853b75cae4d0461b8a1c7ddd4947",
	}
	setNamespaceImageAliases = []string{
		"gcr.io/kpt-fn/set-namespace:v0.4.1",
		"gcr.io/kpt-fn/set-namespace:v0.4",
		"gcr.io/kpt-fn/set-namespace@sha256:f930d9248001fa763799cc81cf2d89bbf83954fc65de0db20ab038a21784f323",
	}
	starlarkImageAliases = []string{
		"gcr.io/kpt-fn/starlark:v0.4.3",
		"gcr.io/kpt-fn/starlark:v0.4",
		"gcr.io/kpt-fn/starlark@sha256:6ba3971c64abcd6c3d93039d45721bb5ab496c7fbbc9ac1e685b11577f368ce0",
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
