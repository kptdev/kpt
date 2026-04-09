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
	fnresult "github.com/kptdev/kpt/pkg/api/fnresult/v1"
	kptfile "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/lib/runneroptions"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestFunctionRunner_Conditions(t *testing.T) {
	ctx := context.Background()
	ctx = printer.WithContext(ctx, printer.New(io.Discard, io.Discard))
	fsys := filesys.MakeFsInMemory()
	celEnv, err := runneroptions.NewCELEnvironment()
	require.NoError(t, err)

	inputNodes := []*yaml.RNode{
		yaml.MustParse("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: app-config"),
	}

	testCases := []struct {
		name      string
		fn        *kptfile.Function
		condition string
		expectRun bool
	}{
		{
			name: "builtin runtime - condition met",
			fn: &kptfile.Function{
				Image: runneroptions.FuncGenPkgContext,
			},
			condition: "resources.exists(r, r.kind == 'ConfigMap')",
			expectRun: true,
		},
		{
			name: "builtin runtime - condition not met",
			fn: &kptfile.Function{
				Image: runneroptions.FuncGenPkgContext,
			},
			condition: "resources.exists(r, r.kind == 'Deployment')",
			expectRun: false,
		},
		{
			name: "executable runtime - condition met",
			fn: &kptfile.Function{
				Exec: "my-exec",
			},
			condition: "resources.size() > 0",
			expectRun: true,
		},
		{
			name: "executable runtime - condition not met",
			fn: &kptfile.Function{
				Exec: "my-exec",
			},
			condition: "resources.size() == 0",
			expectRun: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.fn.Condition = tc.condition
			results := fnresult.NewResultList()
			
			// Mock runner options
			opts := runneroptions.RunnerOptions{
				CELEnvironment: celEnv,
				ResolveToImage: func(image string) string { return image },
			}

			// We use a mock runner to avoid actual execution
			runner, err := NewRunner(ctx, fsys, tc.fn, types.UniquePath("pkg"), results, opts, nil)
			require.NoError(t, err)

			// Override the Run function to track if it's called
			wasRun := false
			runner.filter.Run = func(_ io.Reader, _ io.Writer) error {
				wasRun = true
				return nil
			}

			_, err = runner.Filter(inputNodes)
			require.NoError(t, err)

			assert.Equal(t, tc.expectRun, wasRun, "Run state mismatch for: %s", tc.name)
			assert.Equal(t, !tc.expectRun, runner.WasSkipped(), "Skip state mismatch for: %s", tc.name)
			
			if !tc.expectRun {
				require.NotEmpty(t, results.Items)
				assert.True(t, results.Items[0].Skipped)
				assert.Equal(t, 0, results.Items[0].ExitCode)
			}
		})
	}
}
