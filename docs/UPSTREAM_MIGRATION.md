# Migration from Copied Code to Upstream Dependencies

This document describes the migration from copied third-party code to upstream Kubernetes/kubectl libraries.

## Overview

Prior to v1.0.0, kpt maintained copied and modified versions of code from:
- `sigs.k8s.io/kustomize/kyaml`
- `sigs.k8s.io/kustomize/cmd/config`
- `sigs.k8s.io/cli-utils`

As of v1.0.0, kpt uses upstream versions directly, eliminating maintenance burden and ensuring better compatibility.

## What Was Copied

### thirdparty/kyaml/ **Source**: `sigs.k8s.io/kustomize/kyaml` v0.10.15 **Files**:
- `runfn/runfn.go` - KRM function runner
- `runfn/runfn_test.go` - Tests **Reason for Copy**: Custom modifications for kpt-specific behavior **Migration**: Use upstream `sigs.k8s.io/kustomize/kyaml` v0.21.0+

### thirdparty/cmdconfig/ **Source**: `sigs.k8s.io/kustomize/cmd/config` v0.9.9 **Files**:
- `commands/cmdeval/` - Eval command
- `commands/cmdsink/` - Sink command
- `commands/cmdsource/` - Source command
- `commands/cmdtree/` - Tree command
- `commands/runner/` - Command runner **Reason for Copy**: Integration with kpt command structure **Migration**: Use upstream `sigs.k8s.io/kustomize/kyaml` v0.21.0+ (cmd/config merged into kyaml)

### thirdparty/cli-utils/ **Source**: `sigs.k8s.io/cli-utils` **Files**: Various apply and status utilities **Reason for Copy**: Custom modifications + version pinning **Migration**: Use upstream `sigs.k8s.io/cli-utils` v0.37.2+

## Migration Steps

### Step 1: Update go.mod **Before**:
```go
require (
    sigs.k8s.io/kustomize/kyaml v0.10.15
    sigs.k8s.io/cli-utils v0.26.0
)
``` **After**:
```go
require (
    sigs.k8s.io/kustomize/kyaml v0.21.0
    sigs.k8s.io/cli-utils v0.37.2
)
```

### Step 2: Update Imports **Before**:
```go
import (
    "github.com/kptdev/kpt/thirdparty/kyaml/runfn"
    "github.com/kptdev/kpt/thirdparty/cmdconfig/commands/cmdeval"
)
``` **After**:
```go
import (
    "sigs.k8s.io/kustomize/kyaml/runfn"
    "sigs.k8s.io/kustomize/kyaml/commands/cmdeval"
)
```

### Step 3: Remove thirdparty Directory

```bash
# After migration is complete
rm -rf thirdparty/
```

### Step 4: Update Tests

Update any tests that reference thirdparty code: **Before**:
```go
import "github.com/kptdev/kpt/thirdparty/kyaml/runfn"

func TestRunFn(t *testing.T) {
    r := runfn.RunFns{Path: "testdata"}
    // ...
}
``` **After**:
```go
import "sigs.k8s.io/kustomize/kyaml/runfn"

func TestRunFn(t *testing.T) {
    r := runfn.RunFns{Path: "testdata"}
    // ...
}
```

## API Compatibility

### kyaml API Changes **v0.10.15 → v0.21.0**:

Most APIs remain compatible, but some changes: **PackageBuffer**:
```go
// Still compatible
buff := &kio.PackageBuffer{}
``` **LocalPackageReadWriter**:
```go
// Still compatible
pkg := &kio.LocalPackageReadWriter{
    PackagePath: "path",
    PackageFileName: "Kptfile",
}
``` **RunFns**:
```go
// Still compatible
r := runfn.RunFns{
    Path: "path",
    Functions: []string{"image"},
}
```

### cli-utils API Changes **v0.26.0 → v0.37.2**: **Inventory**:
```go
// Compatible - no changes needed
import "sigs.k8s.io/cli-utils/pkg/common"

label := common.InventoryLabel
``` **Apply**:
```go
// Compatible - minor improvements
import "sigs.k8s.io/cli-utils/pkg/apply"

applier := apply.NewApplier(...)
```

## Benefits of Migration

### 1. Reduced Maintenance

-  No need to manually sync upstream changes
-  Automatic security updates
-  Bug fixes from upstream
-  Less code to maintain

### 2. Better Compatibility

-  Works with latest Kubernetes versions
-  Compatible with other tools using same libraries
-  Consistent behavior across ecosystem

### 3. Community Benefits

-  Contributions benefit entire community
-  Shared testing and validation
-  Better documentation

## Porch Migration

Porch (package orchestration) also needs migration: **Location**: https://github.com/nephio-project/porch **Same Process**:
1. Update go.mod dependencies
2. Update imports
3. Remove copied code
4. Test compatibility **Coordination**: Porch migration should happen in parallel with kpt migration

## Testing After Migration

### Unit Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./pkg/...
go test ./internal/...
```

### Integration Tests

```bash
# Test function rendering
make test-fn-render

# Test function eval
make test-fn-eval

# Test live apply
make test-live-apply
```

### Manual Testing

```bash
# Test package operations
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress
kpt pkg update wordpress/

# Test function operations
kpt fn render wordpress/

# Test live operations
kpt live init wordpress/
kpt live apply wordpress/ --dry-run
```

## Rollback Plan

If issues are found after migration:

### Option 1: Fix Forward

- Identify incompatibility
- Fix in kpt code
- Submit PR to upstream if needed

### Option 2: Temporary Workaround

- Use replace directive in go.mod
- Fork upstream temporarily
- Plan permanent fix **Example**:
```go
// go.mod
replace sigs.k8s.io/kustomize/kyaml => github.com/kptdev/kyaml v0.21.0-kpt.1
```

## Timeline

- **v1.0.0-alpha**: Begin migration
- **v1.0.0-beta**: Complete migration, testing
- **v1.0.0**: Release with upstream dependencies
- **v1.1.0**: Remove thirdparty directory entirely

## Known Issues

### Issue 1: Custom Modifications **Problem**: Some copied code had kpt-specific modifications **Solution**: 
- Contribute changes upstream where possible
- Use composition/wrapping for kpt-specific behavior
- Document any workarounds

### Issue 2: Version Pinning **Problem**: Upstream versions may have breaking changes **Solution**:
- Thorough testing before upgrade
- Pin to specific upstream versions
- Update incrementally

## Contributing Upstream

If you find issues or need features:

1. **Open Issue**: Report in upstream repository
2. **Submit PR**: Contribute fix/feature upstream
3. **Coordinate**: Work with upstream maintainers
4. **Backport**: Use in kpt once merged **Upstream Repositories**:
- kustomize: https://github.com/kubernetes-sigs/kustomize
- cli-utils: https://github.com/kubernetes-sigs/cli-utils

## Verification

After migration, verify:

- [ ] All imports updated
- [ ] No references to thirdparty/
- [ ] All tests pass
- [ ] Integration tests pass
- [ ] Manual testing successful
- [ ] Documentation updated
- [ ] go.mod uses upstream versions
- [ ] No replace directives (unless necessary)

## References

- [kustomize Repository](https://github.com/kubernetes-sigs/kustomize)
- [cli-utils Repository](https://github.com/kubernetes-sigs/cli-utils)
- [Go Modules Documentation](https://go.dev/ref/mod)
- [Semantic Import Versioning](https://research.swtch.com/vgo-import)
