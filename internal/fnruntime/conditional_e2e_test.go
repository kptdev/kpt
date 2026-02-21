// Copyright 2026 The kpt and Nephio Authors
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

package fnruntime

import (
	"context"
	"io"
	"testing"

	"github.com/kptdev/kpt/internal/types"
	fnresultv1 "github.com/kptdev/kpt/pkg/api/fnresult/v1"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestFunctionRunner_ConditionalExecution_E2E tests the complete flow
// of conditional function execution
func TestFunctionRunner_ConditionalExecution_E2E(t *testing.T) {
	ctx := printer.WithContext(context.Background(), printer.New(nil, nil))
	_ = filesys.MakeFsInMemory() // Reserved for future use

	testCases := []struct {
		name           string
		condition      string
		inputResources []string
		shouldExecute  bool
		description    string
	}{
		{
			name:      "condition met - ConfigMap exists",
			condition: `resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "app-config")`,
			inputResources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  env: production`,
			},
			shouldExecute: true,
			description:   "Function should execute when ConfigMap with specific name exists",
		},
		{
			name:      "condition not met - ConfigMap missing",
			condition: `resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "app-config")`,
			inputResources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: other-config
data:
  env: staging`,
			},
			shouldExecute: false,
			description:   "Function should skip when specified ConfigMap doesn't exist",
		},
		{
			name:      "condition met - Deployment count check",
			condition: `resources.filter(r, r.kind == "Deployment").size() > 0`,
			inputResources: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-app
spec:
  replicas: 3`,
			},
			shouldExecute: true,
			description:   "Function should execute when Deployments exist",
		},
		{
			name:      "condition not met - no Deployments",
			condition: `resources.filter(r, r.kind == "Deployment").size() > 0`,
			inputResources: []string{
				`apiVersion: v1
kind: Service
metadata:
  name: web-service`,
			},
			shouldExecute: false,
			description:   "Function should skip when no Deployments exist",
		},
		{
			name:      "always true condition",
			condition: `true`,
			inputResources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: test`,
			},
			shouldExecute: true,
			description:   "Function should always execute with true condition",
		},
		{
			name:      "always false condition",
			condition: `false`,
			inputResources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: test`,
			},
			shouldExecute: false,
			description:   "Function should never execute with false condition",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse input resources
			var input []*yaml.RNode
			for _, resourceYAML := range tc.inputResources {
				rnode, err := yaml.Parse(resourceYAML)
				require.NoError(t, err)
				input = append(input, rnode)
			}

			// Create a mock function that adds an annotation
			functionExecuted := false
			mockFilter := func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
				functionExecuted = true
				// Add an annotation to mark execution
				for _, node := range nodes {
					err := node.PipeE(
						yaml.SetAnnotation("test-annotation", "executed"),
					)
					if err != nil {
						return nil, err
					}
				}
				return nodes, nil
			}

			// Create adapter function to match FunctionFilter.Run signature
			adapterFunc := func(reader io.Reader, writer io.Writer) error {
				// Parse YAML from reader into RNodes
				nodes, err := (&kio.ByteReader{Reader: reader}).Read()
				if err != nil {
					return err
				}

				// Call mockFilter
				resultNodes, err := mockFilter(nodes)
				if err != nil {
					return err
				}

				// Write results back to writer
				return (&kio.ByteWriter{Writer: writer}).Write(resultNodes)
			}

			// Create function runner with condition
			fnResult := &fnresultv1.Result{}
			fnResults := &fnresultv1.ResultList{}

			evaluator, err := NewCELEvaluator(tc.condition)
			require.NoError(t, err)

			runner := &FunctionRunner{
				ctx:              ctx,
				name:             "test-function",
				pkgPath:          types.UniquePath("test"),
				disableCLIOutput: true,
				filter: &runtimeutil.FunctionFilter{
					Run: adapterFunc,
				},
				fnResult:  fnResult,
				fnResults: fnResults,
				opts:      runneroptions.RunnerOptions{},
				condition: tc.condition,
				evaluator: evaluator,
			}

			// Execute the filter
			output, err := runner.Filter(input)
			require.NoError(t, err)

			// Verify function execution based on condition
			if tc.shouldExecute {
				assert.True(t, functionExecuted, tc.description)
				// Verify annotation was added
				annotations := output[0].GetAnnotations()
				annotation := annotations["test-annotation"]
				assert.Equal(t, "executed", annotation)
			} else {
				assert.False(t, functionExecuted, tc.description)
				// Verify output is unchanged (no annotation)
				annotations := output[0].GetAnnotations()
				_, exists := annotations["test-annotation"]
				assert.False(t, exists, "annotation should not exist when function is skipped")
			}
		})
	}
}

// TestFunctionRunner_ConditionalExecution_ComplexConditions tests more advanced CEL expressions
func TestFunctionRunner_ConditionalExecution_ComplexConditions(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name          string
		condition     string
		resources     []string
		shouldExecute bool
	}{
		{
			name:      "multiple conditions with AND",
			condition: `resources.exists(r, r.kind == "ConfigMap") && resources.exists(r, r.kind == "Deployment")`,
			resources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: config`,
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app`,
			},
			shouldExecute: true,
		},
		{
			name:      "check nested field",
			condition: `resources.exists(r, r.kind == "Deployment" && r.spec.replicas > 2)`,
			resources: []string{
				`apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 5`,
			},
			shouldExecute: true,
		},
		{
			name:      "check data field in ConfigMap",
			condition: `resources.exists(r, r.kind == "ConfigMap" && r.data.environment == "production")`,
			resources: []string{
				`apiVersion: v1
kind: ConfigMap
metadata:
  name: env-config
data:
  environment: production
  region: us-west`,
			},
			shouldExecute: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var input []*yaml.RNode
			for _, resourceYAML := range tc.resources {
				rnode, err := yaml.Parse(resourceYAML)
				require.NoError(t, err)
				input = append(input, rnode)
			}

			evaluator, err := NewCELEvaluator(tc.condition)
			require.NoError(t, err)

			result, err := evaluator.EvaluateCondition(ctx, input)
			require.NoError(t, err)
			assert.Equal(t, tc.shouldExecute, result)
		})
	}
}
