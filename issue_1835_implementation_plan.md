## Implementation Plan: Auto-create package directory in `kpt pkg init`

### Issue
**#1835**: `kpt pkg init` should create package directory if it doesn't exist

### Current Behavior
- Command fails with error: `{directory} does not exist`
- Users must manually create directory before running `kpt pkg init`

### Desired Behavior
- Command automatically creates directory if it doesn't exist
- Works with nested paths (e.g., `path/to/package`)
- Maintains backward compatibility with existing directories

---

## Implementation Steps

### 1. Code Changes

#### File: `pkg/kptpkg/init.go`
**Location**: `Initialize()` method, lines ~73-79

**Current Code:**
```go
up := string(p.UniquePath)
if !fsys.Exists(string(p.UniquePath)) {
    return errors.Errorf("%s does not exist", p.UniquePath)
}
```

**New Code:**
```go
pr := printer.FromContextOrDie(ctx)

up := string(p.UniquePath)
if !fsys.Exists(string(p.UniquePath)) {
    pr.Printf("creating package directory %s\n", opts.RelPath)
    if err := fsys.MkdirAll(up); err != nil {
        return errors.Errorf("failed to create directory %s: %w", p.UniquePath, err)
    }
}
```

**Changes:**
- Move `pr` initialization before directory check
- Replace error with `MkdirAll()` call
- Add user-friendly message
- Handle creation errors gracefully

#### File: `commands/pkg/init/cmdinit_test.go`
**Location**: `TestCmd_failNotExists()` test, lines ~172-178

**Current Test:**
```go
func TestCmd_failNotExists(t *testing.T) {
    // Expects error when directory doesn't exist
}
```

**New Test:**
```go
func TestCmd_AutoCreateDir(t *testing.T) {
    // Verifies directory is created automatically
    // Validates Kptfile is generated correctly
}
```

---

### 2. Testing Strategy

#### Unit Tests
- ✅ Test auto-creation of new directory
- ✅ Test existing directory (no "creating" message)
- ✅ Test nested directory creation
- ✅ Test current directory (`.`)
- ✅ Test default to current directory (no args)

#### Manual Testing
```bash
# Test 1: New directory
kpt pkg init my-package --description "test"

# Test 2: Nested directory
kpt pkg init path/to/package --description "test"

# Test 3: Existing directory
mkdir existing-dir
kpt pkg init existing-dir --description "test"

# Test 4: Current directory
mkdir test-dir && cd test-dir
kpt pkg init . --description "test"
```

#### Expected Output
```
creating package directory my-package
writing my-package/Kptfile
writing my-package/README.md
writing my-package/package-context.yaml
```

---

### 3. Edge Cases Handled

| Scenario | Behavior |
|----------|----------|
| Directory doesn't exist | Create automatically |
| Directory exists | Skip creation, proceed normally |
| Nested path | Create all parent directories |
| Permission denied | Return error with clear message |
| Invalid path | Return filesystem error |

---

### 4. Backward Compatibility

**No Breaking Changes:**
- Existing workflows continue to work
- Users who create directories first see no difference
- Only new behavior: auto-creation when directory missing

---

### 5. Documentation Updates

**Files to Update:**
- Command help text (already correct: "init [DIR]")
- Examples in documentation
- Release notes mentioning new behavior

---

### 6. Validation Checklist

- [x] Code compiles without errors
- [x] All existing tests pass
- [x] New test added and passing
- [x] `go vet` passes
- [x] Manual testing completed
- [x] No diagnostics/linting errors
- [x] Backward compatibility verified
- [x] Error handling tested
- [x] Commit message follows conventions
- [x] Signed-off-by added (DCO)

---

### 7. Commit Details

**Branch**: `feat/pkg-init-auto-create-dir`

**Commit Message:**
```
feat: Auto-create package directory in kpt pkg init

- Automatically create package directory if it doesn't exist
- Update test to verify auto-creation behavior
- Add user-friendly message when creating directory

Fixes #1835

Signed-off-by: Surbhi <agarwalsurbhi1807@gmail.com>
```

**Files Changed:**
- `pkg/kptpkg/init.go` (7 insertions, 2 deletions)
- `commands/pkg/init/cmdinit_test.go` (19 insertions, 6 deletions)

---

### 8. Next Steps

1. Create PR from `feat/pkg-init-auto-create-dir` branch
2. Link to issue #1835
3. Wait for CI checks to pass
4. Address review comments if any
5. Merge after approval

---

## Technical Details

### Why `MkdirAll()` instead of `Mkdir()`?
- Supports nested paths: `kpt pkg init path/to/package`
- Creates all parent directories automatically
- Idempotent: doesn't fail if directory exists

### Why move `pr` initialization?
- Need printer to log "creating directory" message
- Must be initialized before directory check
- No functional impact, just reordering

### Error Handling
- Filesystem errors (permissions, disk full) are propagated
- Clear error messages for debugging
- Maintains existing error handling patterns

---

## Risk Assessment

**Risk Level**: Low

**Reasons:**
- Simple, focused change
- Well-tested with multiple scenarios
- Backward compatible
- Follows existing code patterns
- No external dependencies added
