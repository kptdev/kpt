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
	"fmt"

	fnresultv1 "github.com/GoogleContainerTools/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/GoogleContainerTools/kpt/pkg/fn"
	"github.com/GoogleContainerTools/kpt/porch/engine/pkg/internal"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewPlaceholderFunctionRunner() fn.FunctionRunner {
	return &runner{}
}

type runner struct {
}

var _ fn.FunctionRunner = &runner{}

func (e *runner) NewRunner(ctx context.Context, fn *kptfilev1.Function, opts fn.RunnerOptions) (kio.Filter, error) {
	return &filter{
		ctx: ctx,
		fn:  *fn,
		rl:  opts.ResultList,
	}, nil
}

type filter struct {
	ctx context.Context
	fn  kptfilev1.Function
	rl  *fnresultv1.ResultList
}

var _ kio.Filter = &filter{}

func (r *filter) Filter(items []*yaml.RNode) ([]*yaml.RNode, error) {
	rl := &framework.ResourceList{
		Items:   items,
		Results: []*framework.Result{},
	}

	if r.fn.ConfigMap != nil {
		if cm, err := NewConfigMap(r.fn.ConfigMap); err != nil {
			return nil, fmt.Errorf("cannot create config map: %w", err)
		} else {
			rl.FunctionConfig = cm
		}
	}

	if err := internal.Eval(r.fn.Image, rl); err != nil {
		return nil, fmt.Errorf("function evaluation failed; %w", err)
	}

	// Return error on error
	if rl.Results.ExitCode() != 0 {
		return nil, rl.Results
	}

	return rl.Items, nil
}

func NewConfigMap(data map[string]string) (*yaml.RNode, error) {
	node := yaml.NewMapRNode(&data)
	if node == nil {
		return nil, nil
	}
	// create a ConfigMap only for configMap config
	configMap := yaml.MustParse(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: function-input
data: {}
`)
	if err := configMap.PipeE(yaml.SetField("data", node)); err != nil {
		return nil, err
	}
	return configMap, nil
}
