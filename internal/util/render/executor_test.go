// Copyright 2022,2025-2026 The kpt Authors
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

package render

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kptdev/kpt/internal/fnruntime"
	"github.com/kptdev/kpt/internal/pkg"
	"github.com/kptdev/kpt/internal/types"
	fnresult "github.com/kptdev/kpt/pkg/api/fnresult/v1"
	kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"
	"github.com/kptdev/kpt/pkg/kptfile/kptfileutil"
	"github.com/kptdev/kpt/pkg/printer"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const rootString = "/root"
const subPkgString = "/root/subpkg"

func TestPathRelToRoot(t *testing.T) {
	tests := []struct {
		name         string
		rootPath     string
		subPkgPath   string
		resourcePath string
		expected     string
		errString    string
	}{
		{
			name:         "root package with non absolute path",
			rootPath:     "tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("root package path %q must be absolute", "tmp"),
		},
		{
			name:         "subpackage with non absolute path",
			rootPath:     "/tmp",
			subPkgPath:   "tmp/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage path %q must be absolute", "tmp/a"),
		},
		{
			name:         "resource in a subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "c.yaml",
			expected:     "a/c.yaml",
		},
		{
			name:         "resource exists in a deeply nested subpackage",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a/b/c",
			resourcePath: "c.yaml",
			expected:     "a/b/c/c.yaml",
		},
		{
			name:         "resource exists in a sub dir with same name as sub package",
			rootPath:     "/tmp",
			subPkgPath:   "/tmp/a",
			resourcePath: "a/c.yaml",
			expected:     "a/a/c.yaml",
		},
		{
			name:         "subpackage is not a descendant of root package",
			rootPath:     "/tmp",
			subPkgPath:   "/a",
			resourcePath: "c.yaml",
			expected:     "",
			errString:    fmt.Sprintf("subpackage %q is not a descendant of %q", "/a", "/tmp"),
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.name, func(t *testing.T) {
			newPath, err := pathRelToRoot(tc.rootPath,
				tc.subPkgPath, tc.resourcePath)
			assert.Equal(t, newPath, tc.expected)
			if tc.errString != "" {
				assert.Contains(t, err.Error(), tc.errString)
			}
		})
	}
}

func TestMergeWithInput(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		selectedInput string
		output        string
		expected      string
	}{
		{
			name: "simple input",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			selectedInput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: staging
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
spec:
  replicas: 3
`,
		},
		{
			name: "complex example with generation, transformation and deletion of resource",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    internal.config.k8s.io/kpt-resource-id: "1"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
  annotations:
    internal.config.k8s.io/kpt-resource-id: "2"
`,
			selectedInput: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-1
  annotations:
    internal.config.k8s.io/kpt-resource-id: "1"
`,
			output: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  namespace: staging # transformed
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1 # generated resource
kind: Deployment
metadata:
  name: nginx-deployment-3
`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-0
  namespace: staging # transformed
  annotations:
    internal.config.k8s.io/kpt-resource-id: "0"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment-2
  annotations:
    internal.config.k8s.io/kpt-resource-id: "2"
---
apiVersion: apps/v1 # generated resource
kind: Deployment
metadata:
  name: nginx-deployment-3
`,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			output, err := kio.ParseAll(tc.output)
			assert.NoError(t, err)
			selectedInput, err := kio.ParseAll(tc.selectedInput)
			assert.NoError(t, err)
			input, err := kio.ParseAll(tc.input)
			assert.NoError(t, err)
			result := fnruntime.MergeWithInput(output, selectedInput, input)
			actual, err := kio.StringAll(result)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func setupRendererTest(t *testing.T, renderBfs bool) (*Renderer, *bytes.Buffer, context.Context) {
	var outputBuffer bytes.Buffer
	ctx := context.Background()
	ctx = printer.WithContext(ctx, printer.New(&outputBuffer, &outputBuffer))

	mockFileSystem := filesys.MakeFsInMemory()

	rootPkgPath := rootString
	err := mockFileSystem.Mkdir(rootPkgPath)
	assert.NoError(t, err)

	subPkgPath := subPkgString
	err = mockFileSystem.Mkdir(subPkgPath)
	assert.NoError(t, err)

	childPkgPath := "/root/subpkg/child"
	err = mockFileSystem.Mkdir(subPkgPath)
	assert.NoError(t, err)

	siblingPkgPath := "/root/sibling"
	err = mockFileSystem.Mkdir(subPkgPath)
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(rootPkgPath, "Kptfile"), fmt.Appendf(nil, `
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
  annotations:
    kpt.dev/bfs-rendering: %t
`, renderBfs))
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(subPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub-package
`))
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(siblingPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sibling-package
`))
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(childPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: child-package
`))
	assert.NoError(t, err)

	renderer := &Renderer{
		PkgPath:        rootPkgPath,
		ResultsDirPath: "/results",
		FileSystem:     mockFileSystem,
	}

	return renderer, &outputBuffer, ctx
}

func TestRenderer_Execute_RenderOrder(t *testing.T) {
	tests := []struct {
		name          string
		renderBfs     bool
		expectedOrder func(output string) bool
	}{
		{
			name:      "Use hydrateBfsOrder with renderBfs true",
			renderBfs: true,
			expectedOrder: func(output string) bool {
				rootIndex := strings.Index(output, `Package: "root"`)            // First
				siblingIndex := strings.Index(output, `Package: "root/sibling"`) // Second
				return rootIndex < siblingIndex
			},
		},
		{
			name:      "Use default hydrate with renderBfs false",
			renderBfs: false,
			expectedOrder: func(output string) bool {
				siblingIndex := strings.Index(output, `Package: "root/sibling"`) // First
				rootIndex := strings.Index(output, `Package: "root"`)            // Fourth
				return rootIndex > siblingIndex
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			renderer, outputBuffer, ctx := setupRendererTest(t, tc.renderBfs)

			fnResults, err := renderer.Execute(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, fnResults)
			assert.Equal(t, 0, len(fnResults.Items))

			output := outputBuffer.String()
			assert.True(t, tc.expectedOrder(output))
		})
	}
}

func TestHydrate_ErrorCases(t *testing.T) {
	mockFileSystem := filesys.MakeFsInMemory()

	// Create a mock root package
	rootPath := rootString
	err := mockFileSystem.Mkdir(rootPath)
	assert.NoError(t, err)

	// Add a Kptfile to the root package
	err = mockFileSystem.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`))
	assert.NoError(t, err)

	root, err := newPkgNode(mockFileSystem, rootPath, nil)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       root,
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFileSystem,
	}

	t.Run("Cycle Detection in hydrate", func(t *testing.T) {
		// Add the root package to the hydration context in a "Hydrating" state to simulate a cycle
		hctx.pkgs[root.pkg.UniquePath] = &pkgNode{
			pkg:   root.pkg,
			state: Hydrating,
		}

		_, err := hydrate(context.Background(), root, hctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cycle detected in pkg dependencies")
	})

	t.Run("Error in LocalResources", func(t *testing.T) {
		// Simulate an error in LocalResources by creating a package with no Kptfile
		invalidPkgPath := "/invalid"
		err := mockFileSystem.Mkdir(invalidPkgPath)
		assert.NoError(t, err)

		invalidPkgNode, err := newPkgNode(mockFileSystem, invalidPkgPath, nil)
		if err != nil {
			assert.Contains(t, err.Error(), "error reading Kptfile")
			return
		}

		// If no error, proceed to call hydrate (this should not happen in this case)
		_, err = hydrate(context.Background(), invalidPkgNode, hctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read Kptfile")
	})
}

func TestHydrateBfsOrder_ErrorCases(t *testing.T) {
	ctx := printer.WithContext(context.Background(), printer.New(nil, nil))
	mockFileSystem := filesys.MakeFsInMemory()

	rootPkgPath := rootString
	err := mockFileSystem.Mkdir(rootPkgPath)
	assert.NoError(t, err)

	subPkgPath := subPkgString
	err = mockFileSystem.Mkdir(subPkgPath)
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(rootPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
  annotations:
    ktp.dev/bfs-rendering: true
`))
	assert.NoError(t, err)

	err = mockFileSystem.WriteFile(filepath.Join(subPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub-package
`))
	assert.NoError(t, err)

	// Create a mock hydration context
	root, err := newPkgNode(mockFileSystem, rootPkgPath, nil)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       root,
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFileSystem,
	}

	t.Run("Cycle Detection in hydrateBfsOrder", func(t *testing.T) {
		// Add the root package to the hydration context in a "Hydrating" state to simulate a cycle
		hctx.pkgs[root.pkg.UniquePath] = &pkgNode{
			pkg:   root.pkg,
			state: Hydrating,
		}

		_, err := hydrateBfsOrder(ctx, root, hctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cycle detected in pkg dependencies")
	})

	t.Run("Invalid Package State in hydrateBfsOrder", func(t *testing.T) {
		// Add the root package to the hydration context in an invalid state
		hctx.pkgs[root.pkg.UniquePath] = &pkgNode{
			pkg:   root.pkg,
			state: -1, // Invalid state
		}

		_, err := hydrateBfsOrder(ctx, root, hctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "package found in invalid state")
	})

	t.Run("Wet Package State in hydrateBfsOrder would continue", func(t *testing.T) {
		hctx.pkgs[root.pkg.UniquePath] = &pkgNode{
			pkg:   root.pkg,
			state: Wet,
		}

		_, err := hydrateBfsOrder(ctx, root, hctx)
		assert.NoError(t, err)
	})
}

func TestHydrateBfsOrder_RunPipelineError(t *testing.T) {
	ctx := printer.WithContext(context.Background(), printer.New(nil, nil))
	mockFileSystem := filesys.MakeFsInMemory()

	rootPkgPath := rootString
	assert.NoError(t, mockFileSystem.Mkdir(rootPkgPath))

	// Write a Kptfile with an invalid api version
	_ = mockFileSystem.WriteFile(filepath.Join(rootPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/ERROR
kind: Kptfile
metadata:
  name: root-package
  annotations:
    kpt.dev/bfs-rendering: "true"
`))

	p, _ := pkg.New(mockFileSystem, rootPkgPath)
	root := &pkgNode{
		pkg:   p,
		state: Dry,
	}
	hctx := &hydrationContext{
		root:       root,
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFileSystem,
	}

	_, err := hydrateBfsOrder(ctx, root, hctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestRenderer_PrintPipelineExecutionSummary(t *testing.T) {
	tests := []struct {
		name                string
		executedFunctionCnt int
		pkgCount            int
		hydErr              error
		expectedOutput      string
	}{
		{
			name:                "Success with functions executed",
			executedFunctionCnt: 3,
			pkgCount:            2,
			hydErr:              nil,
			expectedOutput:      "Successfully executed 3 function(s) in 2 package(s).\n",
		},
		{
			name:                "Success with no functions",
			executedFunctionCnt: 0,
			pkgCount:            1,
			hydErr:              nil,
			expectedOutput:      "Successfully executed 0 function(s) in 1 package(s).\n",
		},
		{
			name:                "Failure with no functions executed",
			executedFunctionCnt: 0,
			pkgCount:            2,
			hydErr:              fmt.Errorf("pipeline error"),
			expectedOutput:      "Failed to execute any functions in 2 package(s).\n",
		},
		{
			name:                "Partial execution with some functions executed",
			executedFunctionCnt: 2,
			pkgCount:            3,
			hydErr:              fmt.Errorf("pipeline error"),
			expectedOutput:      "Partially executed 2 function(s) in 3 package(s).\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var outputBuffer bytes.Buffer
			pr := printer.New(&outputBuffer, &outputBuffer)

			renderer := &Renderer{}

			hctx := hydrationContext{
				executedFunctionCnt: tc.executedFunctionCnt,
				pkgs:                make(map[types.UniquePath]*pkgNode, tc.pkgCount),
			}

			// Populate pkgs map with dummy entries
			for i := 0; i < tc.pkgCount; i++ {
				hctx.pkgs[types.UniquePath(fmt.Sprintf("/pkg%d", i))] = &pkgNode{}
			}

			renderer.printPipelineExecutionSummary(pr, hctx, tc.hydErr)

			assert.Equal(t, tc.expectedOutput, outputBuffer.String())
		})
	}
}

func TestUpdateRenderStatus_Success(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))

	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}
	hctx.pkgs[rootPkg.UniquePath] = &pkgNode{pkg: rootPkg}

	updateRenderStatus(hctx, nil)

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)
	assert.Len(t, rootKf.Status.Conditions, 1)
	assert.Equal(t, kptfilev1.ConditionTypeRendered, rootKf.Status.Conditions[0].Type)
	assert.Equal(t, kptfilev1.ConditionTrue, rootKf.Status.Conditions[0].Status)
	assert.Equal(t, kptfilev1.ReasonRenderSuccess, rootKf.Status.Conditions[0].Reason)
}

func TestUpdateRenderStatus_Failure(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))

	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}
	hctx.pkgs[rootPkg.UniquePath] = &pkgNode{pkg: rootPkg}

	updateRenderStatus(hctx, fmt.Errorf("set-annotations failed: some error"))

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)
	assert.Len(t, rootKf.Status.Conditions, 1)
	assert.Equal(t, kptfilev1.ConditionFalse, rootKf.Status.Conditions[0].Status)
	assert.Equal(t, kptfilev1.ReasonRenderFailed, rootKf.Status.Conditions[0].Reason)
	assert.Contains(t, rootKf.Status.Conditions[0].Message, "set-annotations failed")
}

func TestUpdateRenderStatus_ReplacesExistingCondition(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))

	// Kptfile with an existing Rendered condition from a previous run
	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
status:
  conditions:
  - type: Rendered
    status: "False"
    reason: RenderFailed
    message: "old error"
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}
	hctx.pkgs[rootPkg.UniquePath] = &pkgNode{pkg: rootPkg}

	updateRenderStatus(hctx, nil)

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)
	assert.Len(t, rootKf.Status.Conditions, 1)
	assert.Equal(t, kptfilev1.ConditionTrue, rootKf.Status.Conditions[0].Status)
	assert.Equal(t, kptfilev1.ReasonRenderSuccess, rootKf.Status.Conditions[0].Reason)
	assert.Empty(t, rootKf.Status.Conditions[0].Message)
}

func TestUpdateRenderStatus_OnlyUpdatesRootKptfile(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))

	subPkgPath := subPkgString
	assert.NoError(t, mockFS.Mkdir(subPkgPath))

	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))
	assert.NoError(t, mockFS.WriteFile(filepath.Join(subPkgPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: sub-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)
	subPkg, err := pkg.New(mockFS, subPkgPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}
	hctx.pkgs[rootPkg.UniquePath] = &pkgNode{pkg: rootPkg}
	hctx.pkgs[subPkg.UniquePath] = &pkgNode{pkg: subPkg}

	updateRenderStatus(hctx, nil)

	// Root should have the condition
	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)
	assert.Len(t, rootKf.Status.Conditions, 1)
	assert.Equal(t, kptfilev1.ConditionTrue, rootKf.Status.Conditions[0].Status)

	// Subpackage should NOT have any condition
	subKf, err := kptfileutil.ReadKptfile(mockFS, subPkgPath)
	assert.NoError(t, err)
	assert.True(t, subKf.Status == nil || len(subKf.Status.Conditions) == 0)
}

func TestBuildRenderStatus_NoSteps(t *testing.T) {
	hctx := &hydrationContext{}
	rs := buildRenderStatus(hctx, nil)
	assert.Nil(t, rs)
}

func TestBuildRenderStatus_SuccessWithMutationSteps(t *testing.T) {
	hctx := &hydrationContext{
		mutationSteps: []kptfilev1.PipelineStepResult{
			{Image: "set-namespace:v1", ExitCode: 0},
			{Image: "set-annotations:v1", ExitCode: 0},
		},
	}
	rs := buildRenderStatus(hctx, nil)
	assert.NotNil(t, rs)
	assert.Len(t, rs.MutationSteps, 2)
	assert.Empty(t, rs.ValidationSteps)
	assert.Empty(t, rs.ErrorSummary)
}

func TestBuildRenderStatus_FailureWithErrorSummary(t *testing.T) {
	hctx := &hydrationContext{
		mutationSteps: []kptfilev1.PipelineStepResult{
			{Image: "set-namespace:v1", ExitCode: 0},
			{Image: "bad-image:v1", ExitCode: 1},
		},
		validationSteps: []kptfilev1.PipelineStepResult{
			{Image: "gatekeeper:latest", ExecutionError: "image not found"},
		},
	}
	rs := buildRenderStatus(hctx, fmt.Errorf("pipeline failed"))
	assert.NotNil(t, rs)
	assert.Contains(t, rs.ErrorSummary, "bad-image:v1: exit code 1")
	assert.Contains(t, rs.ErrorSummary, "gatekeeper:latest: image not found")
}

func TestBuildRenderStatus_UsesNameForErrorSummary(t *testing.T) {
	hctx := &hydrationContext{
		mutationSteps: []kptfilev1.PipelineStepResult{
			{Name: "my-step", Image: "img:v1", ExitCode: 1},
		},
	}
	rs := buildRenderStatus(hctx, fmt.Errorf("fail"))
	assert.Equal(t, "my-step: exit code 1", rs.ErrorSummary)
}

func TestBuildRenderStatus_UsesExecPathForErrorSummary(t *testing.T) {
	hctx := &hydrationContext{
		mutationSteps: []kptfilev1.PipelineStepResult{
			{ExecPath: "/usr/bin/my-fn", ExecutionError: "not found"},
		},
	}
	rs := buildRenderStatus(hctx, fmt.Errorf("fail"))
	assert.Equal(t, "/usr/bin/my-fn: not found", rs.ErrorSummary)
}

func TestCaptureStepResult_FromFnResults(t *testing.T) {
	fnResults := fnresult.NewResultList()
	fnResults.Items = append(fnResults.Items, fnresult.Result{
		Image:    "gatekeeper:latest",
		ExitCode: 1,
		Stderr:   "validation failed",
		Results: framework.Results{
			{Message: "banned key found", Severity: framework.Error,
				ResourceRef: &yaml.ResourceIdentifier{
					TypeMeta: yaml.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
					NameMeta: yaml.NameMeta{Name: "my-cm", Namespace: "default"},
				},
				File: &framework.File{Path: "resources.yaml", Index: 2},
			},
			{Message: "missing label", Severity: framework.Warning},
		},
	})

	fn := kptfilev1.Function{Name: "validate", Image: "gatekeeper:latest"}
	step := captureStepResult(fn, fnResults, 0)

	assert.Equal(t, "validate", step.Name)
	assert.Equal(t, "gatekeeper:latest", step.Image)
	assert.Equal(t, 1, step.ExitCode)
	assert.Equal(t, "validation failed", step.Stderr)
	assert.Len(t, step.Results, 2)

	// First result — error with full resource ref and file
	assert.Equal(t, "banned key found", step.Results[0].Message)
	assert.Equal(t, "error", step.Results[0].Severity)
	assert.Equal(t, "v1", step.Results[0].ResourceRef.APIVersion)
	assert.Equal(t, "ConfigMap", step.Results[0].ResourceRef.Kind)
	assert.Equal(t, "my-cm", step.Results[0].ResourceRef.Name)
	assert.Equal(t, "default", step.Results[0].ResourceRef.Namespace)
	assert.Equal(t, "resources.yaml", step.Results[0].File.Path)
	assert.Equal(t, 2, step.Results[0].File.Index)

	// Second result — warning, no resource ref
	assert.Equal(t, "missing label", step.Results[1].Message)
	assert.Equal(t, "warning", step.Results[1].Severity)
	assert.Nil(t, step.Results[1].ResourceRef)

	// ErrorResults should only contain the error-severity item
	assert.Len(t, step.ErrorResults, 1)
	assert.Equal(t, "banned key found", step.ErrorResults[0].Message)
}

func TestCaptureStepResult_NoNewItems(t *testing.T) {
	fnResults := fnresult.NewResultList()
	fn := kptfilev1.Function{Image: "set-namespace:v1"}
	step := captureStepResult(fn, fnResults, 0)

	assert.Equal(t, "set-namespace:v1", step.Image)
	assert.Equal(t, 0, step.ExitCode)
	assert.Empty(t, step.Stderr)
	assert.Nil(t, step.Results)
	assert.Nil(t, step.ErrorResults)
}

func TestExecutionErrorStep(t *testing.T) {
	fns := []kptfilev1.Function{
		{Name: "my-fn", Image: "bad-image:v1", Exec: ""},
	}
	step := executionErrorStep(fns, fmt.Errorf("pull access denied"))

	assert.Equal(t, "my-fn", step.Name)
	assert.Equal(t, "bad-image:v1", step.Image)
	assert.Equal(t, "pull access denied", step.ExecutionError)
	assert.Equal(t, 0, step.ExitCode)
	assert.Nil(t, step.Results)
}

func TestExecutionErrorStep_EmptyFns(t *testing.T) {
	step := executionErrorStep(nil, fmt.Errorf("no functions"))
	assert.Empty(t, step.Name)
	assert.Empty(t, step.Image)
	assert.Empty(t, step.ExecPath)
	assert.Equal(t, "no functions", step.ExecutionError)
}

func TestFrameworkResultsToItems_Nil(t *testing.T) {
	items := frameworkResultsToItems(nil)
	assert.Nil(t, items)
}

func TestFrameworkResultsToItems_WithFieldRef(t *testing.T) {
	results := framework.Results{
		{
			Message:  "wrong value",
			Severity: framework.Error,
			Field: &framework.Field{
				Path:          ".spec.replicas",
				CurrentValue:  "invalid",
				ProposedValue: 3,
			},
		},
	}
	items := frameworkResultsToItems(results)
	assert.Len(t, items, 1)
	assert.Equal(t, ".spec.replicas", items[0].Field.Path)
	assert.Equal(t, "invalid", items[0].Field.CurrentValue)
	assert.Equal(t, "3", items[0].Field.ProposedValue)
}

func TestFrameworkResultsToItems_NilFieldValues(t *testing.T) {
	results := framework.Results{
		{
			Message:  "field info",
			Severity: framework.Info,
			Field: &framework.Field{
				Path: ".spec.replicas",
			},
		},
	}
	items := frameworkResultsToItems(results)
	assert.Len(t, items, 1)
	assert.Equal(t, ".spec.replicas", items[0].Field.Path)
	assert.Empty(t, items[0].Field.CurrentValue)
	assert.Empty(t, items[0].Field.ProposedValue)
}

func TestUpdateRenderStatus_WritesRenderStatus(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))
	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
		mutationSteps: []kptfilev1.PipelineStepResult{
			{Image: "set-namespace:v1", ExitCode: 0},
		},
		validationSteps: []kptfilev1.PipelineStepResult{
			{Image: "gatekeeper:latest", ExitCode: 1, Stderr: "failed"},
		},
	}

	updateRenderStatus(hctx, fmt.Errorf("validation failed"))

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)

	// Condition should be set
	assert.Len(t, rootKf.Status.Conditions, 1)
	assert.Equal(t, kptfilev1.ConditionFalse, rootKf.Status.Conditions[0].Status)

	// RenderStatus should be populated
	rs := rootKf.Status.RenderStatus
	assert.NotNil(t, rs)
	assert.Len(t, rs.MutationSteps, 1)
	assert.Equal(t, "set-namespace:v1", rs.MutationSteps[0].Image)
	assert.Len(t, rs.ValidationSteps, 1)
	assert.Equal(t, "gatekeeper:latest", rs.ValidationSteps[0].Image)
	assert.Contains(t, rs.ErrorSummary, "gatekeeper:latest: exit code 1")
}

func TestUpdateRenderStatus_NilRenderStatusWhenNoSteps(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))
	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}

	updateRenderStatus(hctx, nil)

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status)
	assert.Nil(t, rootKf.Status.RenderStatus)
}

func TestUpdateRenderStatus_ClearsPreviousRenderStatus(t *testing.T) {
	mockFS := filesys.MakeFsInMemory()
	rootPath := rootString
	assert.NoError(t, mockFS.Mkdir(rootPath))
	assert.NoError(t, mockFS.WriteFile(filepath.Join(rootPath, "Kptfile"), []byte(`
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: root-package
`)))

	rootPkg, err := pkg.New(mockFS, rootPath)
	assert.NoError(t, err)

	// First render: failure with steps
	hctx := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
		mutationSteps: []kptfilev1.PipelineStepResult{
			{Image: "bad:v1", ExitCode: 1},
		},
	}
	updateRenderStatus(hctx, fmt.Errorf("fail"))

	rootKf, err := kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.NotNil(t, rootKf.Status.RenderStatus)

	// Second render: success with no steps (empty pipeline)
	hctx2 := &hydrationContext{
		root:       &pkgNode{pkg: rootPkg},
		pkgs:       map[types.UniquePath]*pkgNode{},
		fileSystem: mockFS,
	}
	updateRenderStatus(hctx2, nil)

	rootKf, err = kptfileutil.ReadKptfile(mockFS, rootPath)
	assert.NoError(t, err)
	assert.Nil(t, rootKf.Status.RenderStatus)
}

func TestStepName(t *testing.T) {
	assert.Equal(t, "my-step", stepName(kptfilev1.PipelineStepResult{Name: "my-step", Image: "img:v1"}))
	assert.Equal(t, "img:v1", stepName(kptfilev1.PipelineStepResult{Image: "img:v1"}))
	assert.Equal(t, "/usr/bin/fn", stepName(kptfilev1.PipelineStepResult{ExecPath: "/usr/bin/fn"}))
	assert.Equal(t, "", stepName(kptfilev1.PipelineStepResult{}))
}

func TestPkgNode_ClearAnnotationsOnMutFailure(t *testing.T) {
	tests := []struct {
		name                      string
		inputYAML                 string
		hasNonRenderingAnnotation bool
	}{
		{
			name: "clears all migration annotations",
			inputYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    config.k8s.io/id: "123"
    internal.config.kubernetes.io/annotations-migration-resource-id: "456"
    internal.config.kubernetes.io/id: "789"
    internal.config.k8s.io/kpt-resource-id: "abc"
    other.annotation: "keep"`,
			hasNonRenderingAnnotation: true,
		},
		{
			name: "handles resources without migration annotations",
			inputYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  annotations:
    other.annotation: "keep"`,
			hasNonRenderingAnnotation: true,
		},
		{
			name: "handles resources with no annotations",
			inputYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test`,
			hasNonRenderingAnnotation: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := kio.ParseAll(tc.inputYAML)
			assert.NoError(t, err)

			clearAnnotationsOnMutFailure(nodes)

			for _, node := range nodes {
				annotations := node.GetAnnotations()
				assert.NotContains(t, annotations, "config.k8s.io/id")
				assert.NotContains(t, annotations, "internal.config.kubernetes.io/annotations-migration-resource-id")
				assert.NotContains(t, annotations, "internal.config.kubernetes.io/id")
				assert.NotContains(t, annotations, "internal.config.k8s.io/kpt-resource-id")
				// Verify other.annotation is preserved after clearing
				if tc.hasNonRenderingAnnotation {
					assert.Contains(t, annotations, "other.annotation")
				}
			}
		})
	}
}
