# Migration Guide to kpt v1.0.0

This guide helps you migrate from earlier versions of kpt to v1.0.0.

## Overview

kpt v1.0.0 is the first stable release with guaranteed API stability. This release includes:

-  Stable v1 APIs for all core types
-  Semantic versioning for kpt, SDK, and functions
-  Proper version reporting across all architectures
-  Use of upstream Kubernetes/kubectl types (no more copied code)
-  Clear backward compatibility guarantees

## Breaking Changes

### 1. ResourceGroup API: v1alpha1 → v1 **What Changed**: ResourceGroup API has been promoted from `v1alpha1` to `v1`. **Migration**: **Before** (v1alpha1):
```yaml
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: inventory
  namespace: default
``` **After** (v1):
```yaml
apiVersion: kpt.dev/v1
kind: ResourceGroup
metadata:
  name: inventory
  namespace: default
``` **Action Required**:
- Update all `resourcegroup.yaml` files to use `apiVersion: kpt.dev/v1`
- Update code imports from `pkg/api/resourcegroup/v1alpha1` to `pkg/api/resourcegroup/v1` **Backward Compatibility**: kpt v1.0.0 can still read v1alpha1 ResourceGroups, but will write v1.

### 2. Version Command Output **What Changed**: `kpt version` now shows semantic version instead of git commit hash. **Before**:
```bash
$ kpt version
a1b2c3d
``` **After**:
```bash
$ kpt version
kpt version: v1.0.0
``` **Action Required**: Update any scripts that parse version output.

### 3. Removed Deprecated Kptfile Versions **What Changed**: Support for very old Kptfile versions (v1alpha1, v1alpha2) has been removed. **Action Required**: 
- If you have packages with old Kptfile versions, update them first
- Use kpt v0.39.x to migrate old packages to v1 format
- See: https://kpt.dev/installation/migration

### 4. Upstream Dependencies **What Changed**: kpt now uses upstream Kubernetes/kubectl libraries instead of copied code. **Impact**: 
- Better compatibility with Kubernetes ecosystem
- Faster security updates
- Reduced maintenance burden **Action Required**: None for users. Function developers should update imports if using internal kpt packages.

## Non-Breaking Changes

### 1. New Versioning Documentation

- Added `docs/VERSIONING.md` with complete versioning policy
- Clear semantic versioning for all components
- Compatibility matrix for kpt, SDK, and functions

### 2. Improved Error Messages

- Better error messages for version mismatches
- Clear migration instructions in error output

### 3. Multi-Architecture Support

- Verified version command works on all platforms:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)

## Migration Steps

### Step 1: Check Current Version

```bash
kpt version
```

### Step 2: Backup Your Packages

```bash
# Backup your kpt packages
cp -r my-package my-package-backup
```

### Step 3: Update ResourceGroup Files

```bash
# Find all resourcegroup files
find . -name "resourcegroup.yaml" -type f

# Update apiVersion in each file
sed -i 's/apiVersion: kpt.dev\/v1alpha1/apiVersion: kpt.dev\/v1/g' */resourcegroup.yaml
```

### Step 4: Update Kptfiles (if needed)

```bash
# Check Kptfile versions
find . -name "Kptfile" -exec grep "apiVersion:" {} \;

# All should show: apiVersion: kpt.dev/v1
# If you see v1alpha1 or v1alpha2, update the package
```

### Step 5: Test Your Packages

```bash
# Render to verify everything works
kpt fn render my-package

# If using live commands, test apply in dry-run mode
kpt live apply my-package --dry-run
```

### Step 6: Update CI/CD Pipelines

Update any scripts that:
- Parse `kpt version` output
- Check for specific kpt versions
- Use deprecated APIs **Example CI/CD Update**: **Before**:
```bash
# Old version check
VERSION=$(kpt version)
if [ "$VERSION" != "a1b2c3d" ]; then
  echo "Wrong version"
fi
``` **After**:
```bash
# New version check
VERSION=$(kpt version | grep -oP 'v\d+\.\d+\.\d+')
if [ "$VERSION" != "v1.0.0" ]; then
  echo "Wrong version"
fi
```

## For Function Developers

### Update SDK Dependency **Before** (go.mod):
```go
require (
    github.com/kptdev/krm-functions-sdk/go/fn v0.x.x
)
``` **After** (go.mod):
```go
require (
    github.com/kptdev/krm-functions-sdk/go/fn v1.0.2
)
```

### Update ResourceGroup Imports **Before**:
```go
import (
    rgfilev1alpha1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1alpha1"
)

func example() {
    gvk := rgfilev1alpha1.ResourceGroupGVK()
}
``` **After**:
```go
import (
    rgfilev1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1"
)

func example() {
    gvk := rgfilev1.ResourceGroupGVK()
}
```

### Version Your Functions

Ensure your functions follow semantic versioning:

```dockerfile
# In your function Dockerfile
LABEL version="v1.0.0"
LABEL sdk-version="v1.0.2"
```

## Troubleshooting

### Issue: "Kptfile has an old version" **Error**:
```
Error: Kptfile at "my-package/Kptfile" has an old version (v1alpha1) of the Kptfile schema.
``` **Solution**:
1. Use kpt v0.39.x to migrate the package
2. Or manually update the Kptfile apiVersion to `kpt.dev/v1`
3. See: https://kpt.dev/installation/migration

### Issue: "ResourceGroup version mismatch" **Error**:
```
Warning: ResourceGroup uses deprecated v1alpha1 API
``` **Solution**:
Update resourcegroup.yaml:
```bash
sed -i 's/apiVersion: kpt.dev\/v1alpha1/apiVersion: kpt.dev\/v1/g' resourcegroup.yaml
```

### Issue: Version command shows "unknown" **Cause**: Development build without proper version tag **Solution**:
- Use official releases from https://github.com/kptdev/kpt/releases
- Or build with proper version: `make build VERSION=v1.0.0`

### Issue: Function compatibility **Error**:
```
Function requires SDK v1.x.x but kpt types are incompatible
``` **Solution**:
1. Update function to use SDK v1.0.2+
2. Ensure function uses kpt v1 types
3. Rebuild and republish function

## Rollback Plan

If you encounter issues with v1.0.0:

### Option 1: Rollback to Previous Version

```bash
# Download previous version
# See: https://github.com/kptdev/kpt/releases

# Restore backup
rm -rf my-package
cp -r my-package-backup my-package
```

### Option 2: Use Compatibility Mode

kpt v1.0.0 maintains backward compatibility with v1alpha1 ResourceGroups for reading.

## Getting Help

- **Issues**: https://github.com/kptdev/kpt/issues
- **Discussions**: https://github.com/kptdev/kpt/discussions
- **Slack**: #kpt channel on Kubernetes Slack
- **Documentation**: https://kpt.dev

## Checklist

Use this checklist to ensure smooth migration:

- [ ] Backed up all kpt packages
- [ ] Updated ResourceGroup files to v1
- [ ] Verified all Kptfiles use kpt.dev/v1
- [ ] Updated CI/CD scripts for new version format
- [ ] Tested package rendering with `kpt fn render`
- [ ] Tested live commands with `--dry-run`
- [ ] Updated function dependencies (if applicable)
- [ ] Reviewed versioning documentation
- [ ] Informed team members about changes

## Timeline

- **v1.0.0 Release**: April 2026
- **v1alpha1 Deprecation**: Immediate (still readable, but not written)
- **v1alpha1 Removal**: v2.0.0 (estimated 12+ months)

## What's Next?

After migrating to v1.0.0:

1. **Explore New Features**: Check release notes for new capabilities
2. **Update Documentation**: Update your team's documentation
3. **Monitor Releases**: Watch for v1.x.x updates with new features
4. **Contribute**: Help improve kpt by contributing feedback and code

Thank you for using kpt! 
