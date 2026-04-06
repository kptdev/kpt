# Backward Compatibility Policy

## Overview

kpt v1.0.0 and later follow strict backward compatibility guarantees to ensure stable, production-ready usage.

## Compatibility Guarantees

### Within Major Version (v1.x.x)

All v1.x.x releases are backward compatible with v1.0.0: **Guaranteed Compatible**:
- Kptfile v1 format
- ResourceGroup v1 API
- Function result v1 API
- CLI command structure
- Flag names and behavior
- Configuration file formats **Additive Changes Only**:
- New commands can be added
- New flags can be added (with defaults)
- New API fields can be added (optional)
- New function types can be added **Not Allowed**:
- Removing commands
- Removing flags
- Changing flag behavior
- Removing API fields
- Changing API field types
- Breaking existing workflows

### Across Major Versions

Major version changes (v1 → v2) may include breaking changes:

- API changes that are not backward compatible
- Removal of deprecated features
- Changes to core behavior **Migration Support**:
- Deprecation warnings in v1.x.x before removal in v2.0.0
- Migration guides provided
- Minimum 6-month deprecation period

## API Stability Levels

### Stable (v1) **Status**: Production-ready, fully supported **APIs**:
- `pkg/api/kptfile/v1`
- `pkg/api/fnresult/v1`
- `pkg/api/resourcegroup/v1` **Guarantees**:
- No breaking changes within v1.x.x
- Security patches backported
- Bug fixes provided
- New optional fields may be added **Example**:
```go
import kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"

// This import will work for all v1.x.x releases
```

### Deprecated (v1alpha1) **Status**: Maintained for compatibility, will be removed in v2.0.0 **APIs**:
- `pkg/api/resourcegroup/v1alpha1` (use v1 instead) **Guarantees**:
- Read support maintained in v1.x.x
- Write operations use v1 format
- Deprecation warnings shown
- Removed in v2.0.0 **Migration Path**:
```go
// Old (deprecated)
import rgfilev1alpha1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1alpha1"

// New (stable)
import rgfilev1 "github.com/kptdev/kpt/pkg/api/resourcegroup/v1"
```

## Function Compatibility

### SDK Compatibility **Rule**: Functions built with SDK v1.x.x work with kpt v1.x.x **Matrix**:
| Function SDK | kpt v1.0.x | kpt v1.1.x | kpt v1.2.x |
|--------------|------------|------------|------------|
| v1.0.x       |          |          |          |
| v1.1.x       |          |          |          |
| v1.2.x       |          |          |          |

### Type Compatibility **Rule**: Functions using kpt types don't need version bumps unless types change **When to Bump Function Version**:
-  Function logic changes
-  Function behavior changes
-  New features added
-  kpt types remain unchanged (no bump needed) **Example**:
```go
// Function using kpt types
import kptfilev1 "github.com/kptdev/kpt/pkg/api/kptfile/v1"

func Transform(rl *fn.ResourceList) error {
    // Uses kptfilev1.KptFile
    // No version bump needed if only kpt updates
}
```

## Testing Compatibility

### Automated Testing

kpt maintains automated compatibility tests:

1. **API Compatibility Tests**
   - Verify v1 APIs don't change
   - Test old packages with new kpt versions
   - Validate function compatibility

2. **Multi-Version Tests**
   - Test kpt v1.x with SDK v1.y
   - Test old functions with new kpt
   - Test new functions with old kpt (within v1)

3. **Architecture Tests**
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64)

### Manual Testing Checklist

Before each release:

- [ ] Old packages render with new kpt
- [ ] Old functions work with new kpt
- [ ] Version command works on all architectures
- [ ] Deprecated APIs still readable
- [ ] Migration guides tested
- [ ] Backward compatibility tests pass

## Deprecation Process

### Phase 1: Announcement (v1.x.0)

- Feature marked as deprecated in code
- Deprecation notice in release notes
- Documentation updated
- Migration guide provided **Example**:
```go
// Deprecated: Use NewFunction instead. Will be removed in v2.0.0.
func OldFunction() {}
```

### Phase 2: Warning Period (v1.x.0 to v1.y.0)

- Deprecation warnings shown in CLI
- Warnings in logs
- Documentation shows alternatives
- Minimum 6 months or 1 minor version **Example CLI Warning**:
```bash
$ kpt live apply
Warning: ResourceGroup v1alpha1 is deprecated. Use v1 instead.
See: https://kpt.dev/migration
```

### Phase 3: Removal (v2.0.0)

- Feature removed in next major version
- Migration guide available
- Clear error messages if old format used

## Version Detection

### Runtime Version Checking

```go
import "github.com/kptdev/kpt/run"

// Check kpt version at runtime
version := run.Version()
if !isCompatible(version, "v1.0.0") {
    return fmt.Errorf("requires kpt v1.0.0+, got %s", version)
}
```

### Package Version Checking

```yaml
# In Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
info:
  description: Requires kpt v1.0.0+
```

## Breaking Change Examples

###  Allowed in Minor Version (v1.x.0) **Adding Optional Field**:
```go
// v1.0.0
type Config struct {
    Name string
}

// v1.1.0 - OK: new optional field
type Config struct {
    Name        string
    Description string `yaml:"description,omitempty"`
}
``` **Adding New Command**:
```bash
# v1.0.0
kpt pkg get
kpt pkg update

# v1.1.0 - OK: new command
kpt pkg get
kpt pkg update
kpt pkg sync  # New!
```

###  Not Allowed in Minor Version **Removing Field**:
```go
// v1.0.0
type Config struct {
    Name string
    OldField string
}

// v1.1.0 - NOT OK: breaks compatibility
type Config struct {
    Name string
    // OldField removed - BREAKING!
}
``` **Changing Flag Behavior**:
```bash
# v1.0.0
kpt fn render --output=stdout  # prints to stdout

# v1.1.0 - NOT OK: changes behavior
kpt fn render --output=stdout  # writes to file - BREAKING!
```

## Compatibility Testing in CI

### Example GitHub Actions

```yaml
name: Compatibility Tests

on: [push, pull_request]

jobs:
  test-compatibility:
    strategy:
      matrix:
        kpt-version: [v1.0.0, v1.1.0, v1.2.0]
        sdk-version: [v1.0.0, v1.0.2, v1.1.0]
    
    steps:
      - name: Test Function Compatibility
        run: |
          # Test function built with SDK ${{ matrix.sdk-version }}
          # Works with kpt ${{ matrix.kpt-version }}
          kpt fn eval --image=my-func:${{ matrix.sdk-version }}
```

## Support Timeline

| Version | Release Date | Full Support | Security Only | End of Life |
|---------|--------------|--------------|---------------|-------------|
| v1.0.x  | Apr 2026     | Current      | -             | -           |
| v0.39.x | 2023         | Ended        | 6 months      | Oct 2026    |

## Reporting Compatibility Issues

If you find a compatibility issue:

1. **Check Version**: Ensure you're using compatible versions
2. **Review Docs**: Check migration guides and release notes
3. **Open Issue**: Report at https://github.com/kptdev/kpt/issues
4. **Provide Details**:
   - kpt version
   - SDK version (if applicable)
   - Function versions
   - Reproduction steps **Issue Template**:
```markdown
## Compatibility Issue **kpt version**: v1.0.0 **SDK version**: v1.0.2 **Function**: my-func:v1.0.0 **Expected**: Function should work **Actual**: Error: ... **Steps to reproduce**:
1. ...
2. ...
```

## Best Practices

### For Package Authors

1. **Use Stable APIs**: Always use v1 APIs, not alpha
2. **Test Upgrades**: Test packages with new kpt versions
3. **Document Requirements**: Specify minimum kpt version
4. **Follow Semver**: Version your packages semantically

### For Function Developers

1. **Pin SDK Version**: Use specific SDK version in go.mod
2. **Test Compatibility**: Test with multiple kpt versions
3. **Document Dependencies**: Specify kpt and SDK requirements
4. **Version Functions**: Follow semantic versioning

### For Users

1. **Stay Updated**: Use latest v1.x.x release
2. **Read Release Notes**: Check for deprecations
3. **Test Before Upgrading**: Test in non-production first
4. **Report Issues**: Help improve compatibility

## References

- [Semantic Versioning](https://semver.org/)
- [Kubernetes API Versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning)
- [Go Module Compatibility](https://go.dev/blog/module-compatibility)
- [kpt Versioning Policy](./VERSIONING.md)
- [Migration Guide](./MIGRATION_V1.md)
