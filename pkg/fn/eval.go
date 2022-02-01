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

package fn

import (
	"context"

	"github.com/GoogleContainerTools/kpt/internal/types"
	fnresult "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

type EvalOptions struct {
	// FnResultList stores the result of the function evaluation
	FnResultList fnresult.ResultList
}

type Evaluator interface {
	Eval(ctx context.Context, pkg filesys.FileSystem, fn v1.Function, opts EvalOptions) error
	NewRunner(ctx context.Context, fn *v1.Function, pkgPath types.UniquePath, opts EvalOptions) (kio.Filter, error)
}
