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
	"sigs.k8s.io/kustomize/kyaml/fn/runtime/runtimeutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// TestFunctionRunner_ConditionalExecution_E2E tests the complete flow
// of conditional function execution using the shared CEL environment.
func TestFunctionRunner_ConditionalExecution_E2E(t *testing.T) {
	ctx := printer.WithContext(context.Background(), printer.New(nil, nil))

	celEnv, err := runneroptions.NewCELEnvironment()
	require.NoError(t, err)

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
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app-config\ndata:\n  env: production",
			},
			shouldExecute: true,
			description:   "Function should execute when ConfigMap with specific name exists",
		},
		{
			name:      "condition not met - ConfigMap missing",
			condition: `resources.exists(r, r.kind == "ConfigMap" && r.metadata.name == "app-config")`,
			inputResources: []string{
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: other-config\ndata:\n  env: staging",
			},
			shouldExecute: false,
			description:   "Function should skip when specified ConfigMap doesn't exist",
		},
		{
			name:      "condition met - Deployment count check",
			condition: `resources.filter(r, r.kind == "Deployment").size() > 0`,
			inputResources: []string{
				"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: web-app\nspec:\n  replicas: 3",
			},
			shouldExecute: true,
			description:   "Function should execute when Deployments exist",
		},
		{
			name:      "condition not met - no Deployments",
			condition: `resources.filter(r, r.kind == "Deployment").size() > 0`,
			inputResources: []string{
				"apiVersion: v1\nkind: Service\nmetadata:\n  name: web-service",
			},
			shouldExecute: false,
			description:   "Function should skip when no Deployments exist",
		},
		{
			name:          "always true condition",
			condition:     `true`,
			inputResources: []string{"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test"},
			shouldExecute: true,
			description:   "Function should always execute with true condition",
		},
		{
			name:          "always false condition",
			condition:     `false`,
			inputResources: []string{"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test"},
			shouldExecute: false,
			description:   "Function should never execute with false condition",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var input []*yaml.RNode
			for _, resourceYAML := range tc.inputResources {
				rnode, err := yaml.Parse(resourceYAML)
				require.NoError(t, err)
				input = append(input, rnode)
			}

			functionExecuted := false
			mockFilter := func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
				functionExecuted = true
				for _, node := range nodes {
					if err := node.PipeE(yaml.SetAnnotation("test-annotation", "executed")); err != nil {
						return nil, err
					}
				}
				return nodes, nil
			}

			adapterFunc := func(reader io.Reader, writer io.Writer) error {
				nodes, err := (&kio.ByteReader{Reader: reader}).Read()
				if err != nil {
					return err
				}
				resultNodes, err := mockFilter(nodes)
				if err != nil {
					return err
				}
				return (&kio.ByteWriter{Writer: writer}).Write(resultNodes)
			}

			fnResult := &fnresultv1.Result{}
			fnResults := &fnresultv1.ResultList{}

			runner := &FunctionRunner{
				ctx:              ctx,
				name:             "test-function",
				pkgPath:          types.UniquePath("test"),
				disableCLIOutput: true,
				filter:           &runtimeutil.FunctionFilter{Run: adapterFunc},
				fnResult:         fnResult,
				fnResults:        fnResults,
				opts:             runneroptions.RunnerOptions{},
				condition:        tc.condition,
				celEnv:           celEnv,
			}

			output, err := runner.Filter(input)
			require.NoError(t, err)

			if tc.shouldExecute {
				assert.True(t, functionExecuted, tc.description)
				assert.Equal(t, "executed", output[0].GetAnnotations()["test-annotation"])
			} else {
				assert.False(t, functionExecuted, tc.description)
				_, exists := output[0].GetAnnotations()["test-annotation"]
				assert.False(t, exists, "annotation should not exist when function is skipped")
			}
		})
	}
}

// TestFunctionRunner_ConditionalExecution_ComplexConditions tests more advanced CEL expressions
// directly against the shared CEL environment.
func TestFunctionRunner_ConditionalExecution_ComplexConditions(t *testing.T) {
	ctx := context.Background()

	celEnv, err := runneroptions.NewCELEnvironment()
	require.NoError(t, err)

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
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: config",
				"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app",
			},
			shouldExecute: true,
		},
		{
			name:      "check nested field",
			condition: `resources.exists(r, r.kind == "Deployment" && r.spec.replicas > 2)`,
			resources: []string{
				"apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: app\nspec:\n  replicas: 5",
			},
			shouldExecute: true,
		},
		{
			name:      "check data field in ConfigMap",
			condition: `resources.exists(r, r.kind == "ConfigMap" && r.data.environment == "production")`,
			resources: []string{
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: env-config\ndata:\n  environment: production\n  region: us-west",
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

			result, err := celEnv.EvaluateCondition(ctx, tc.condition, input)
			require.NoError(t, err)
			assert.Equal(t, tc.shouldExecute, result)
		})
	}
}
