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

package kptops

import (
	"context"
	"fmt"
	"io"

	kptfilev1 "github.com/kptdev/kpt/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/fn"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// mockFnRuntime is a test FunctionRuntime that captures the items passed to the function.
type mockFnRuntime struct {
	// capturedItems records the resource names received in items for each invocation
	capturedItems [][]string
	// failOnInvocation specifies which invocation (0-based) should return an error.
	// Only used when shouldFail is true.
	failOnInvocation int
	// shouldFail enables failure on the specified invocation.
	shouldFail bool
}

func (m *mockFnRuntime) GetRunner(_ context.Context, _ *kptfilev1.Function) (fn.FunctionRunner, error) {
	return &mockFnRunner{runtime: m}, nil
}

// mockFnRunner captures input items and passes them through unmodified.
type mockFnRunner struct {
	runtime *mockFnRuntime
}

func (r *mockFnRunner) Run(reader io.Reader, writer io.Writer) error {
	// Use ByteReadWriter to parse the ResourceList wire format
	rw := &kio.ByteReadWriter{
		Reader:                reader,
		Writer:                writer,
		KeepReaderAnnotations: true,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		WrappingKind:          kio.ResourceListKind,
	}
	nodes, err := rw.Read()
	if err != nil {
		return err
	}
	// Record the names of items received
	var names []string
	for _, node := range nodes {
		names = append(names, node.GetName())
	}
	r.runtime.capturedItems = append(r.runtime.capturedItems, names)

	// Fail if this is the designated failing invocation
	invocation := len(r.runtime.capturedItems) - 1
	if r.runtime.shouldFail && r.runtime.failOnInvocation == invocation {
		return fmt.Errorf("intentional failure on invocation %d", invocation)
	}

	// Pass through unmodified
	return rw.Write(nodes)
}

// mutatingMockFnRuntime is a test FunctionRuntime that mutates fn-config on the first invocation.
type mutatingMockFnRuntime struct {
	// capturedFnConfigData records the functionConfig's data map for each invocation.
	capturedFnConfigData []map[string]string
	// capturedItems records the resource names received in items for each invocation.
	capturedItems [][]string
}

func (m *mutatingMockFnRuntime) GetRunner(_ context.Context, _ *kptfilev1.Function) (fn.FunctionRunner, error) {
	return &mutatingMockFnRunner{runtime: m}, nil
}

// mutatingMockFnRunner mutates any ConfigMap named "fn-config" in items on its
// first invocation by adding "mutated-by: first-fn" to the data map. Subsequent
// invocations pass through unmodified but capture the functionConfig data.
type mutatingMockFnRunner struct {
	runtime *mutatingMockFnRuntime
}

func (r *mutatingMockFnRunner) Run(reader io.Reader, writer io.Writer) error {
	rw := &kio.ByteReadWriter{
		Reader:                reader,
		Writer:                writer,
		KeepReaderAnnotations: true,
		WrappingAPIVersion:    kio.ResourceListAPIVersion,
		WrappingKind:          kio.ResourceListKind,
	}
	nodes, err := rw.Read()
	if err != nil {
		return err
	}

	// Record items
	var names []string
	for _, node := range nodes {
		names = append(names, node.GetName())
	}
	r.runtime.capturedItems = append(r.runtime.capturedItems, names)

	// Capture functionConfig data
	if rw.FunctionConfig != nil {
		r.runtime.capturedFnConfigData = append(r.runtime.capturedFnConfigData, rw.FunctionConfig.GetDataMap())
	} else {
		r.runtime.capturedFnConfigData = append(r.runtime.capturedFnConfigData, nil)
	}

	// On first invocation, mutate any ConfigMap named "fn-config" in items
	// by adding a data key. This simulates a mutator that modifies a resource
	// which happens to be another function's fn-config.
	invocation := len(r.runtime.capturedItems) - 1
	if invocation == 0 {
		for _, node := range nodes {
			if node.GetName() == "fn-config" && node.GetKind() == "ConfigMap" {
				dataMap := node.GetDataMap()
				if dataMap == nil {
					dataMap = map[string]string{}
				}
				dataMap["mutated-by"] = "first-fn"
				node.SetDataMap(dataMap)
				break
			}
		}
	}

	return rw.Write(nodes)
}
