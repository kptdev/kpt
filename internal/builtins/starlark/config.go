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

	"github.com/kptdev/krm-functions-catalog/functions/go/starlark/third_party/sigs.k8s.io/kustomize/kyaml/fn/runtime/starlark"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	// Source is a required field for providing a starlark script inline.
	Source string `json:"source" yaml:"source"`
	// Params are the parameters in key-value pairs format.
	Params map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
}

func (sr *Run) Config(fnCfg *fn.KubeObject) error {
	switch {
	case fnCfg.IsEmpty():
		return fmt.Errorf("FunctionConfig is missing. Expect `ConfigMap` or `StarlarkRun`")
	case fnCfg.IsGVK("", configMapAPIVersion, configMapKind):
		cm := &corev1.ConfigMap{}
		if err := fnCfg.As(cm); err != nil {
			return err
		}
		// Convert ConfigMap to StarlarkRun
		sr.Name = cm.Name
		sr.Namespace = cm.Namespace
		sr.Params = map[string]interface{}{}
		for k, v := range cm.Data {
			if k == sourceKey {
				sr.Source = v
			}
			sr.Params[k] = v
		}
	case fnCfg.IsGVK(starlarkRunGroup, starlarkRunVersion, starlarkRunKind),
		fnCfg.IsGVK(starlarkRunGroup, starlarkRunVersion, "Run"):
		if err := fnCfg.As(sr); err != nil {
			return err
		}
	default:
		return fmt.Errorf("`functionConfig` must be either %v or %v, but we got: %v",
			schema.FromAPIVersionAndKind(configMapAPIVersion, configMapKind).String(),
			schema.FromAPIVersionAndKind(starlarkRunAPIVersion, starlarkRunKind).String(),
			schema.FromAPIVersionAndKind(fnCfg.GetAPIVersion(), fnCfg.GetKind()).String())
	}

	// Defaulting
	if sr.Name == "" {
		sr.Name = defaultProgramName
	}
	// Validation
	if sr.Source == "" {
		return fmt.Errorf("`source` must not be empty")
	}
	return nil
}

func (sr *Run) Transform(rl *fn.ResourceList) error {
	var transformedObjects []*fn.KubeObject
	var nodes []*yaml.RNode

	fcRN, err := yaml.Parse(rl.FunctionConfig.String())
	if err != nil {
		return err
	}
	for _, obj := range rl.Items {
		objRN, err := yaml.Parse(obj.String())
		if err != nil {
			return err
		}
		nodes = append(nodes, objRN)
	}

	starFltr := &starlark.SimpleFilter{
		Name:           sr.Name,
		Program:        sr.Source,
		FunctionConfig: fcRN,
	}
	transformedNodes, err := starFltr.Filter(nodes)
	if err != nil {
		return err
	}

	for _, n := range transformedNodes {
		obj, err := fn.ParseKubeObject([]byte(n.MustString()))
		if err != nil {
			return err
		}
		transformedObjects = append(transformedObjects, obj)
	}
	rl.Items = transformedObjects
	return nil
}
