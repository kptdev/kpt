# Implementation Plan: Issue #4388 - Enable skipping pipeline steps based on CEL expressions

**Link**: https://github.com/kptdev/kpt/issues/4388

## Overview

Enable conditional execution of pipeline steps (KRM functions) using CEL (Common Expression Language) expressions that evaluate against the package contents.

## Problem Statement

Currently, kpt package authors need custom KRM functions to conditionally execute pipeline steps based on package contents. This is cumbersome when a simple condition check is needed before running standard functions like `set-annotation`.

## Proposed Solution

Add an optional `condition` field to the `Function` struct in the Pipeline schema that accepts a CEL expression. The pipeline runner will evaluate this expression against the KRM resources, and only execute the function if the expression evaluates to true.

## Technical Design

### 1. Schema Changes

**File**: `pkg/api/kptfile/v1/types.go`

Add a new field to the `Function` struct (line 297):

```go
type Function struct {
    // ... existing fields ...
    
    // Condition is an optional CEL expression that determines whether this
    // function should be executed. The expression has access to the KRM
    // resources in the package and should return a boolean.
    // If omitted or evaluates to true, the function executes normally.
    // Example: "resources.exists(r, r.kind == 'ConfigMap' && r.metadata.name == 'secret-config')"
    Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
    
    // ... remaining fields ...
}
```

### 2. CEL Integration

**New File**: `internal/fnruntime/cel_evaluator.go`

```go
package fnruntime

import (
    "fmt"
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
    "sigs.k8s.io/kustomize/kyaml/yaml"
)

// CELEvaluator evaluates CEL expressions against KRM resources
type CELEvaluator struct {
    env *cel.Env
}

// NewCELevaluator creates a new CEL evaluator with KRM-aware environment
func NewCELEvaluator() (*CELEvaluator, error) {
    env, err := cel.NewEnv(
        cel.Declarations(
            decls.NewVar("resources", decls.NewListType(decls.Dyn)),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create CEL environment: %w", err)
    }
    return &CELEvaluator{env: env}, nil
}

// EvaluateCondition evaluates a CEL expression against input resources
// Returns true if the condition is met, false otherwise
func (e *CELEvaluator) EvaluateCondition(expression string, resources []*yaml.RNode) (bool, error) {
    if expression == "" {
        // Empty condition means always execute
        return true, nil
    }
    
    // Parse the CEL expression
    ast, issues := e.env.Compile(expression)
    if issues != nil && issues.Err() != nil {
        return false, fmt.Errorf("CEL compilation error: %w", issues.Err())
    }
    
    // Create program
    prg, err := e.env.Program(ast)
    if err != nil {
        return false, fmt.Errorf("CEL program creation error: %w", err)
    }
    
    // Convert resources to CEL-compatible format
    resourcesData := convertResourcesToMap(resources)
    
    // Evaluate
    result, _, err := prg.Eval(map[string]interface{}{
        "resources": resourcesData,
    })
    if err != nil {
        return false, fmt.Errorf("CEL evaluation error: %w", err)
    }
    
    // Extract boolean result
    boolResult, ok := result.Value().(bool)
    if !ok {
        return false, fmt.Errorf("CEL expression must return boolean, got %T", result.Value())
    }
    
    return boolResult, nil
}

// convertResourcesToMap converts YAML RNodes to CEL-compatible map structure
func convertResourcesToMap(resources []*yaml.RNode) []interface{} {
    result := make([]interface{}, 0, len(resources))
    for _, resource := range resources {
        // Convert each resource to a map
        resourceMap := make(map[string]interface{})
        // Extract common fields like kind, apiVersion, metadata, spec, etc.
        if kind, err := resource.GetString("kind"); err == nil {
            resourceMap["kind"] = kind
        }
        if apiVersion, err := resource.GetString("apiVersion"); err == nil {
			resourceMap["apiVersion"] = apiVersion
        }
        // Add metadata as a nested map
        if metadata, err := resource.Pipe(yaml.Lookup("metadata")); err == nil && metadata != nil {
            metadataMap := make(map[string]interface{})
            if name, err := metadata.GetString("name"); err == nil {
                metadataMap["name"] = name
            }
            if namespace, err := metadata.GetString("namespace"); err == nil {
                metadataMap["namespace"] = namespace
            }
            // TODO: Add labels, annotations, etc.
            resourceMap["metadata"] = metadataMap
        }
        result = append(result, resourceMap)
    }
    return result
}
```

### 3. Runner Modifications

**File**: `internal/fnruntime/runner.go`

Modify the `Filter` method to check conditions before execution (around line 198):

```go
func (fr *FunctionRunner) Filter(input []*yaml.RNode) (output []*yaml.RNode, err error) {
    // NEW: Check if condition is met before executing
    if fr.condition != "" {
        shouldExecute, err := fr.evaluator.EvaluateCondition(fr.condition, input)
        if err != nil {
            return nil, fmt.Errorf("failed to evaluate condition: %w", err)
        }
        if !shouldExecute {
            pr := printer.FromContextOrDie(fr.ctx)
            pr.Printf("[SKIPPED] %q (condition not met)\n", fr.name)
            // Return input unchanged
            return input, nil
        }
    }
    
    // Existing execution logic...
    pr := printer.FromContextOrDie(fr.ctx)
    // ... rest of the method
}
```

Update the `FunctionRunner` struct to include condition and evaluator:

```go
type FunctionRunner struct {
    ctx              context.Context
    name             string
    pkgPath          types.UniquePath
    disableCLIOutput bool
    filter           *runtimeutil.FunctionFilter
    fnResult         *fnresult.Result
    fnResults        *fnresult.ResultList
    opts             runneroptions.RunnerOptions
    condition        string        // NEW
    evaluator        *CELEvaluator // NEW
}
```

Update `NewRunner` to initialize the CEL evaluator (around line 47):

```go
func NewRunner(
    ctx context.Context,
    fsys filesys.FileSystem,
    f *kptfilev1.Function,
    pkgPath types.UniquePath,
    fnResults *fnresult.ResultList,
    opts runneroptions.RunnerOptions,
    runtime fn.FunctionRuntime,
) (*FunctionRunner, error) {
    // ... existing code ...
    
    // NEW: Create CEL evaluator if condition is specified
    var evaluator *CELEvaluator
    if f.Condition != "" {
        var err error
        evaluator, err = NewCELEvaluator()
        if err != nil {
            return nil, fmt.Errorf("failed to create CEL evaluator: %w", err)
        }
    }
    
    return NewFunctionRunner(ctx, fltr, pkgPath, fnResult, fnResults, opts, f.Condition, evaluator)
}
```

Update `NewFunctionRunner` signature:

```go
func NewFunctionRunner(ctx context.Context,
    fltr *runtimeutil.FunctionFilter,
    pkgPath types.UniquePath,
    fnResult *fnresult.Result,
    fnResults *fnresult.ResultList,
    opts runneroptions.RunnerOptions,
    condition string,          // NEW parameter
    evaluator *CELEvaluator) (*FunctionRunner, error) { // NEW parameter
    // ... existing code ...
    return &FunctionRunner{
        ctx:       ctx,
        name:      name,
        pkgPath:   pkgPath,
        filter:    fltr,
        fnResult:  fnResult,
        fnResults: fnResults,
        opts:      opts,
        condition: condition,    // NEW field
        evaluator: evaluator,    // NEW field
    }, nil
}
```

## Dependencies

Add CEL library to `go.mod`:
```
github.com/google/cel-go v0.20.1
```

## Example Usage

```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: set-annotation
      configMap:
        my-annotation: "value"
      # Only run if there's a ConfigMap named 'feature-flags' with enabled: true
      condition: "resources.exists(r, r.kind == 'ConfigMap' && r.metadata.name == 'feature-flags')"
    
    - image: set-namespace
      configMap:
        namespace: production
      # Only run if no Namespace resource exists yet
      condition: "!resources.exists(r, r.kind == 'Namespace')"
```

## Testing Strategy

### Unit Tests

**File**: `internal/fnruntime/cel_evaluator_test.go`
- Test CEL expression compilation
- Test evaluation with various resource inputs
- Test error handling (invalid expressions, wrong return types)
- Test edge cases (empty resources, nil values)

**File**: `internal/fnruntime/runner_test.go`
- Test function skipping when condition is false
- Test function execution when condition is true
- Test error propagation from CEL evaluator

### E2E Tests

**Directory**: `e2e/testdata/fn-render/conditional-execution/`
- Test package with conditional mutator that should execute
- Test package with conditional mutator that should skip
- Test package with multiple conditional steps
- Test error cases (invalid CEL expressions)

## Implementation Steps

1. ✅ **Research & Planning** (Current step)
   - Understand the codebase structure
   - Design the solution
   - Create implementation plan

2. **Add CEL dependency**
   - Update `go.mod` with CEL library
   - Run `go mod tidy`

3. **Schema Changes**
   - Add `Condition` field to `Function` struct
   - Run code generation if needed

4. **Implement CEL Evaluator**
   - Create `cel_evaluator.go`
   - Implement resource-to-map conversion
   - Write unit tests

5. **Modify Function Runner**
   - Update `FunctionRunner` struct
   - Modify `Filter` method to check conditions
   - Update constructor functions

6. **Documentation**
   - Update Kptfile schema documentation
   - Add examples to user guide
   - Document CEL expression syntax and available variables

7. **Testing**
   - Write unit tests
   - Create E2E test cases
   - Manual testing with real packages

8. **Code Review & Iteration**
   - Submit PR
   - Address review feedback

## Potential Challenges

1. **CEL Expression Complexity**: Need to provide good documentation and examples
2. **Performance**: CEL evaluation should be fast, but need to benchmark with large packages
3. **Error Messages**: Need clear error messages when CEL expressions fail
4. **Backward Compatibility**: Empty/missing condition field should maintain current behavior

## Success Criteria

- [ ] Function with `condition: ""` behaves same as functions without condition
- [ ] Function only executes when condition evaluates to true
- [ ] Clear error messages for invalid CEL expressions
- [ ] Comprehensive unit test coverage (>80%)
- [ ] E2E tests covering common use cases
- [ ] Documentation updated with examples
- [ ] No performance degradation for packages without conditions
- [ ] Backward compatible with existing Kptfiles

## Timeline Estimate

- Research & Planning: 1 day ✅
- Implementation: 3-4 days
- Testing: 2 days
- Documentation: 1 day
- Code Review & Iteration: 2-3 days

**Total**: ~10-12 days

## Next Steps

Ready to start implementation! Would you like me to:
1. Start with adding the CEL dependency?
2. Implement the schema changes first?
3. Create the CEL evaluator module?
