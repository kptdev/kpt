// Copyright 2022, 2025-2026 The kpt Authors
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

package kptops

import (
	"context"

	fnresultv1 "github.com/kptdev/kpt/api/fnresult/v1"
	"github.com/kptdev/kpt/pkg/fn"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/kptdev/kpt/pkg/printer"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

func NewRenderer(runnerOptions runneroptions.RunnerOptions) fn.Renderer {
	return &renderer{runnerOptions: runnerOptions}
}

type renderer struct {
	runnerOptions runneroptions.RunnerOptions
}

var _ fn.Renderer = &renderer{}

func (r *renderer) Render(ctx context.Context, pkg filesys.FileSystem, opts fn.RenderOptions) (*fnresultv1.ResultList, error) {
	// TODO: deal with this
	// if opts.DisplayName != "" && r.runnerOptions.RootDisplayName == "" { //nolint:staticcheck // SA1019
	//	 r.runnerOptions.RootDisplayName = opts.DisplayName //nolint:staticcheck // SA1019
	// }

	rr := Renderer{
		PkgPath:       opts.PkgPath,
		Runtime:       opts.Runtime,
		FileSystem:    pkg,
		RunnerOptions: r.runnerOptions,
	}
	return rr.Execute(printer.WithContext(ctx, printer.NewKlogPrinter()))
}
