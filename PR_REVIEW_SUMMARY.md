# PR #4391 Review Feedback Summary

## Review from mozesl-nokia (6 hours ago)

### 1. Remove unnecessary type alias file
**File:** `internal/fnruntime/celeval.go`
**Issue:** Having a file just for a type alias seems unnecessary
**Action:** Consider removing this file and using `*runneroptions.CELEnvironment` directly in `runner.go`

### 2. Group constants together
**File:** `pkg/lib/runneroptions/celenv.go` (lines 28-31)
**Issue:** Constants should be grouped in a single `const ()` block
**Suggested change:**
```go
const (
    celCheckFrequency = 100
    // celCostLimit gives about .1 seconds of CPU time for the evaluation to run
    celCostLimit = 1000000
)
```

### 3. Avoid panic in InitDefaults
**File:** `pkg/lib/runneroptions/runneroptions.go` (lines 70-76)
**Issue:** `InitDefaults` panics on CEL environment initialization failure, which crashes the process
**Action:** Move CEL environment creation out of `InitDefaults` and let callers handle the error
**Recommendation:** Callers of `InitDefaults` should try to create CEL environment separately and return errors gracefully

### 4. Use `any` instead of `interface{}`
**File:** `pkg/lib/runneroptions/celenv.go`
**Issue:** Modern Go style prefers `any` over `interface{}`
**Action:** Replace all `interface{}` with `any`

## Copilot Review Comments (4 days ago)

### 1. Better error message for exec-based functions
**File:** `internal/fnruntime/runner.go` (line 165)
**Issue:** Error uses `f.Image` which is empty for exec-based functions
**Suggested fix:**
```go
name := f.Image
if name == "" {
    name = f.Exec
}
return nil, fmt.Errorf("condition specified for function %q but no CEL environment is configured in RunnerOptions", name)
```

### 2. Fix go.mod dependency
**File:** `go.mod` (line 136)
**Issue:** `k8s.io/apiserver v0.34.1 // indirect` should be a direct dependency since it's imported directly
**Action:** Remove `// indirect` comment - `go mod tidy` will fix this

## Additional Context

### From nagygergo (yesterday):
- Code and tests look good
- Documentation updates needed:
  1. Update https://kpt.dev/reference/schema/kptfile/
  2. Add new chapter to https://kpt.dev/book/04-using-functions/

### Question to clarify:
Should documentation updates be part of this PR or a separate PR?

## Files to Modify

1. `internal/fnruntime/celeval.go` - Consider removing or justify keeping
2. `pkg/lib/runneroptions/celenv.go` - Group constants, use `any` instead of `interface{}`
3. `pkg/lib/runneroptions/runneroptions.go` - Remove panic from `InitDefaults`
4. `internal/fnruntime/runner.go` - Improve error message for exec functions
5. `go.mod` - Fix k8s.io/apiserver dependency marking

## Next Steps

1. Address all review comments from mozesl-nokia
2. Respond to or resolve Copilot comments
3. Run `go mod tidy` to fix dependency issues
4. Clarify documentation approach with maintainers
5. Test all changes locally before pushing
