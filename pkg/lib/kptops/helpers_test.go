// Copyright 2026 The kpt Authors
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

// helpers_test.go provides a lightweight, in-process FunctionRuntime for use
// in unit tests only. The function implementations here are simplified
// versions that avoid the need for a container runtime (Docker/Podman).

package kptops

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"

	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

// testFunctions maps image references to in-process function implementations.
// Test Kptfiles must use these exact image strings for the runtime to resolve them.
var testFunctions = map[string]framework.ResourceListProcessorFunc{
	"ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1.5":    setLabels,
	"ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1": setNamespace,
}

// runtime is a test-only FunctionRuntime that resolves functions from testFunctions.
type runtime struct{}

var _ FunctionRuntime = &runtime{}

func (e *runtime) GetRunner(ctx context.Context, funct *kptfilev1.Function) (fn.FunctionRunner, error) {
	processor, ok := testFunctions[funct.Image]
	if !ok {
		return nil, &fn.NotFoundError{Function: *funct}
	}
	return &runner{ctx: ctx, processor: processor}, nil
}

func (e *runtime) Close() error { return nil }

type runner struct {
	ctx       context.Context
	processor framework.ResourceListProcessorFunc
}

var _ fn.FunctionRunner = &runner{}

func (fr *runner) Run(r io.Reader, w io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                r,
		Writer:                w,
		KeepReaderAnnotations: true,
	}
	return framework.Execute(fr.processor, rw)
}

// setLabels is a simplified test-only implementation of the set-labels KRM function.
func setLabels(rl *framework.ResourceList) error {
	if rl.FunctionConfig == nil {
		return nil
	}
	if !validGVK(rl.FunctionConfig, "v1", "ConfigMap") {
		return errors.New("invalid set-labels function config; expected v1/ConfigMap")
	}
	labels := rl.FunctionConfig.GetDataMap()
	for _, n := range rl.Items {
		l := n.GetLabels()
		maps.Copy(l, labels)
		if err := n.SetLabels(l); err != nil {
			return err
		}
	}
	return nil
}

// setNamespace is a simplified test-only implementation of the set-namespace KRM function.
func setNamespace(rl *framework.ResourceList) error {
	if rl.FunctionConfig == nil {
		return nil
	}
	if !validGVK(rl.FunctionConfig, "v1", "ConfigMap") {
		return fmt.Errorf("invalid set-namespace function config type: %s/%s; expected v1/ConfigMap",
			rl.FunctionConfig.GetApiVersion(), rl.FunctionConfig.GetKind())
	}
	data := rl.FunctionConfig.GetDataMap()
	if data == nil {
		return nil
	}
	namespace, ok := data["namespace"]
	if !ok {
		return nil
	}
	for _, n := range rl.Items {
		if err := n.SetNamespace(namespace); err != nil {
			return err
		}
	}
	return nil
}

func validGVK(rn *kyaml.RNode, apiVersion, kind string) bool {
	meta, err := rn.GetMeta()
	if err != nil {
		return false
	}
	return meta.APIVersion == apiVersion && meta.Kind == kind
}
