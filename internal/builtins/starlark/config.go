// Copyright 2026 Google LLC
// Modifications Copyright (C) 2025-2026 OpenInfra Foundation Europe
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package starlark

import (
	"fmt"

	starlarkruntime "github.com/kptdev/kpt/internal/builtins/starlark/runtime"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	starlarkRunGroup      = "fn.kpt.dev"
	starlarkRunVersion    = "v1alpha1"
	starlarkRunAPIVersion = starlarkRunGroup + "/" + starlarkRunVersion
	starlarkRunKind       = "StarlarkRun"

	configMapAPIVersion = "v1"
	configMapKind       = "ConfigMap"

	sourceKey = "source"

	defaultProgramName = "starlark-function-run"
)

type Run struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`
	Source            string                 `json:"source" yaml:"source"`
	Params            map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
}

func (sr *Run) Config(fnCfg *yaml.RNode) error {
	if fnCfg == nil {
		return fmt.Errorf("FunctionConfig is missing. Expect `ConfigMap`, `StarlarkRun`, or `Run`")
	}

	meta, err := fnCfg.GetMeta()
	if err != nil {
		return fmt.Errorf("reading functionConfig metadata: %w", err)
	}

	apiVersion := meta.APIVersion
	kind := meta.Kind

	switch {
	case apiVersion == configMapAPIVersion && kind == configMapKind:
		cm := &corev1.ConfigMap{}
		if err := fnCfg.YNode().Decode(cm); err != nil {
			return err
		}
		sr.Name = cm.Name
		sr.Namespace = cm.Namespace
		sr.Params = map[string]interface{}{}
		for k, v := range cm.Data {
			if k == sourceKey {
				sr.Source = v
			}
			sr.Params[k] = v
		}

	case (apiVersion == starlarkRunAPIVersion && kind == starlarkRunKind) ||
		(apiVersion == starlarkRunAPIVersion && kind == "Run"):
		if err := fnCfg.YNode().Decode(sr); err != nil {
			return err
		}

	default:
		return fmt.Errorf("`functionConfig` must be either %v, %v, or %v but we got: %v",
			schema.FromAPIVersionAndKind(configMapAPIVersion, configMapKind).String(),
			schema.FromAPIVersionAndKind(starlarkRunAPIVersion, starlarkRunKind).String(),
			schema.FromAPIVersionAndKind(starlarkRunAPIVersion, "Run").String(),
			schema.FromAPIVersionAndKind(apiVersion, kind).String())
	}

	if sr.Name == "" {
		sr.Name = defaultProgramName
	}
	if sr.Source == "" {
		return fmt.Errorf("`source` must not be empty")
	}
	return nil
}

func (sr *Run) Transform(rl *framework.ResourceList) error {
	starFltr := &starlarkruntime.Filter{
		Name:    sr.Name,
		Program: sr.Source,
	}
	return rl.Filter(starFltr)
}
