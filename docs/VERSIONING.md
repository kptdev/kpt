# kpt Versioning and API Stability

## Overview

kpt follows [Semantic Versioning 2.0.0](https://semver.org/) for all its components. This document describes the versioning strategy for kpt, the SDK, and the function catalog.

## Version Format

All kpt components use semantic versioning in the format: `vMAJOR.MINOR.PATCH`

- **MAJOR**: Incremented for incompatible API changes
- **MINOR**: Incremented for backwards-compatible functionality additions
- **PATCH**: Incremented for backwards-compatible bug fixes

## Component Versioning

### 1. kpt Core Tool

The kpt CLI tool is versioned independently and follows semantic versioning. **Current Stable Version**: v1.0.0 **Version Command**:
```bash
kpt version
``` **Compatibility Promise**:
- v1.x.x releases maintain backward compatibility with v1.0.0
- Breaking changes will only be introduced in v2.0.0
- All v1 APIs are stable and production-ready

### 2. kpt SDK (krm-functions-sdk)

The SDK for building KRM functions is versioned separately from kpt. **Repository**: `github.com/kptdev/krm-functions-sdk` **Current Version**: v1.0.2 **Compatibility**:
- SDK v1.x.x is compatible with kpt v1.x.x
- Functions built with SDK v1.x.x work with kpt v1.x.x

### 3. Function Catalog (krm-functions-catalog)

Individual functions in the catalog are versioned independently. **Repository**: `github.com/kptdev/krm-functions-catalog` **Versioning Strategy**:
- Each function has its own semantic version
- Functions specify their SDK version requirements
- Functions are backward compatible within the same major version

## API Stability Levels

### Stable (v1)

APIs marked as v1 are stable and production-ready:
- `pkg/api/kptfile/v1` - Kptfile API
- `pkg/api/fnresult/v1` - Function result API
- `pkg/api/resourcegroup/v1` - ResourceGroup API (promoted from v1alpha1) **Guarantees**:
- No breaking changes within v1.x.x
- Backward compatibility maintained
- Deprecated features will be supported for at least one major version

### Alpha (v1alpha1, v1alpha2)

Alpha APIs are experimental and may change:
- May have bugs
- Support may be dropped without notice
- Not recommended for production use **Migration**: v1alpha1 APIs have been promoted to v1 as of kpt v1.0.0

## Dependency Relationships

```
┌─────────────────────┐
│  kpt CLI (v1.x.x)   │
│  - Core tool        │
└──────────┬──────────┘
           │ uses types from
           ▼
┌─────────────────────┐
│  kpt Types (v1)     │
│  - API definitions  │
└──────────┬──────────┘
           │ used by
           ▼
┌─────────────────────┐
│  SDK (v1.x.x)       │
│  - Function builder │
└──────────┬──────────┘
           │ used by
           ▼
┌─────────────────────┐
│  Functions (v*)     │
│  - Individual funcs │
└─────────────────────┘
```

## Version Compatibility Matrix

| kpt Version | SDK Version | Function Catalog | Notes |
|-------------|-------------|------------------|-------|
| v1.0.0+     | v1.0.0+     | v0.x.x, v1.x.x   | Stable release |
| v0.39.x     | v0.x.x      | v0.x.x           | Legacy (deprecated) |

## Upgrade Guidelines

### Upgrading kpt

```bash
# Check current version
kpt version

# Download latest version
# See https://kpt.dev/installation/
```

### Upgrading Functions

Functions using kpt types don't need version bumps unless:
1. The kpt types API changes (breaking change)
2. The function's own logic changes
3. The SDK version changes with breaking changes

### Breaking Change Policy **When we bump MAJOR version**:
- Incompatible API changes
- Removal of deprecated features
- Changes to core behavior that break existing workflows **When we bump MINOR version**:
- New features added
- New APIs introduced
- Deprecation notices (features still work) **When we bump PATCH version**:
- Bug fixes
- Security patches
- Documentation updates

## Deprecation Policy

1. **Announcement**: Deprecated features are announced in release notes
2. **Grace Period**: Minimum one major version (e.g., deprecated in v1.5.0, removed in v2.0.0)
3. **Warnings**: Deprecation warnings shown in CLI output
4. **Migration Guide**: Documentation provided for migration path

## Version Checking

### In Code

```go
import "github.com/kptdev/kpt/run"

// Access version at runtime
version := run.Version()
```

### In CI/CD

```bash
# Verify minimum version
REQUIRED_VERSION="v1.0.0"
CURRENT_VERSION=$(kpt version | grep -oP 'v\d+\.\d+\.\d+')

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$CURRENT_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "kpt version $CURRENT_VERSION is older than required $REQUIRED_VERSION"
    exit 1
fi
```

## Release Process

1. **Version Tag**: Create git tag with semantic version (e.g., `v1.0.0`)
2. **Build**: Automated build via goreleaser
3. **Test**: Multi-architecture testing (Linux, macOS, Windows on amd64 and arm64)
4. **Publish**: Release to GitHub and container registries
5. **Announce**: Update documentation and announce release

## Support Policy

- **Current Major Version**: Full support (bug fixes, security patches, new features)
- **Previous Major Version**: Security patches only for 6 months after new major release
- **Older Versions**: Community support only

## Questions?

For questions about versioning:
- Open an issue: https://github.com/kptdev/kpt/issues
- Discussions: https://github.com/kptdev/kpt/discussions
- Slack: https://kubernetes.slack.com/channels/kpt

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Kubernetes API Versioning](https://kubernetes.io/docs/reference/using-api/#api-versioning)
- [kpt Installation Guide](https://kpt.dev/installation/)
