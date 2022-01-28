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

package kpt

import (
	"context"

	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/internal"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func NewPlaceholderEvaluator() fn.Evaluator {
	return &evaluator{}
}

type evaluator struct {
}

var _ fn.Evaluator = &evaluator{}

func (e *evaluator) Eval(ctx context.Context, pkg filesys.FileSystem, fn v1.Function, opts fn.EvalOptions) error {
	return internal.Eval(ctx, pkg, fn, opts)
}
