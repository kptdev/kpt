# SDK and Function Catalog Versioning

This document describes the versioning strategy for the kpt SDK and Function Catalog.

## Overview

The kpt ecosystem consists of three independently versioned components:

1. **kpt CLI** - The core tool (this repository)
2. **kpt SDK** - Function development SDK ([krm-functions-sdk](https://github.com/kptdev/krm-functions-sdk))
3. **Function Catalog** - Pre-built functions ([krm-functions-catalog](https://github.com/kptdev/krm-functions-catalog))

## SDK Versioning

### Repository **Location**: https://github.com/kptdev/krm-functions-sdk **Go Module**: `github.com/kptdev/krm-functions-sdk/go/fn`

### Version Strategy

The SDK follows semantic versioning independently from kpt:

- **Major Version**: Breaking API changes in SDK
- **Minor Version**: New SDK features, backward compatible
- **Patch Version**: Bug fixes, no API changes

### Current Version **Stable**: v1.0.2

### Compatibility with kpt

| SDK Version | Compatible kpt Versions | Notes |
|-------------|------------------------|-------|
| v1.0.x      | v1.0.0+                | Stable, production-ready |
| v0.x.x      | v0.39.x                | Legacy, deprecated |

### Using the SDK **In go.mod**:
```go
module github.com/example/my-function

go 1.21

require (
    github.com/kptdev/krm-functions-sdk/go/fn v1.0.2
)
``` **In Function Code**:
```go
package main

import (
    "github.com/kptdev/krm-functions-sdk/go/fn"
)

func main() {
    if err := fn.AsMain(fn.ResourceListProcessorFunc(process)); err != nil {
        os.Exit(1)
    }
}

func process(rl *fn.ResourceList) (bool, error) {
    // Your function logic
    return true, nil
}
```

### SDK Release Process

1. **Version Tag**: Create semantic version tag (e.g., `v1.0.2`)
2. **Release Notes**: Document changes and compatibility
3. **Go Module**: Publish to Go module proxy
4. **Documentation**: Update SDK documentation
5. **Announce**: Notify in kpt channels

### SDK Backward Compatibility **Within v1.x.x**:
-  All v1.x.x versions are compatible
-  New features added as optional
-  Existing APIs remain stable
-  No breaking changes **Across Major Versions**:
- Breaking changes allowed in v2.0.0
- Migration guide provided
- Deprecation period of 6+ months

## Function Catalog Versioning

### Repository **Location**: https://github.com/kptdev/krm-functions-catalog **Container Registry**: `ghcr.io/kptdev/krm-functions-catalog`

### Version Strategy

Each function in the catalog is versioned independently: **Function Versioning**:
- Each function has its own semantic version
- Functions specify SDK version requirements
- Functions are tagged with version in container registry **Example Functions**:
- `set-namespace:v0.4.1`
- `set-labels:v0.1.5`
- `apply-replacements:v0.1.0`

### Function Metadata

Each function should include version metadata: **In Dockerfile**:
```dockerfile
FROM golang:1.21 as builder
WORKDIR /go/src/
COPY . .
RUN CGO_ENABLED=0 go build -o /usr/local/bin/function .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/local/bin/function /usr/local/bin/function

# Version metadata
LABEL version="v1.0.0"
LABEL sdk-version="v1.0.2"
LABEL kpt-min-version="v1.0.0"

ENTRYPOINT ["/usr/local/bin/function"]
``` **In Function Config**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-function-config
  annotations:
    config.kubernetes.io/function: |
      container:
        image: ghcr.io/kptdev/krm-functions-catalog/my-function:v1.0.0
    function.kpt.dev/sdk-version: v1.0.2
    function.kpt.dev/min-kpt-version: v1.0.0
data:
  # Function configuration
```

### Function Compatibility Matrix

| Function Version | SDK Version | kpt Version | Status |
|------------------|-------------|-------------|--------|
| v1.x.x           | v1.0.x      | v1.0.0+     | Stable |
| v0.x.x           | v0.x.x      | v0.39.x     | Legacy |

### Function Release Process

1. **Version Bump**: Update function version in code
2. **Build**: Build container image
3. **Tag**: Tag image with semantic version
4. **Test**: Test with kpt v1.x.x
5. **Publish**: Push to container registry
6. **Document**: Update function documentation
7. **Catalog**: Update function catalog listing

### Function Versioning Rules **When to Bump Function Version**: **Major (v1.0.0 → v2.0.0)**:
- Breaking changes to function behavior
- Incompatible configuration changes
- Removal of features **Minor (v1.0.0 → v1.1.0)**:
- New features added
- New configuration options (optional)
- Backward compatible changes **Patch (v1.0.0 → v1.0.1)**:
- Bug fixes
- Security patches
- Documentation updates **No Version Bump Needed**:
- kpt types update (if backward compatible)
- SDK patch update (if no API changes)
- Internal refactoring

## Dependency Management

### Dependency Chain

```
┌─────────────────────┐
│  Function (v1.x.x)  │
│  - Your function    │
└──────────┬──────────┘
           │ depends on
           ▼
┌─────────────────────┐
│  SDK (v1.x.x)       │
│  - Function builder │
└──────────┬──────────┘
           │ uses types from
           ▼
┌─────────────────────┐
│  kpt Types (v1)     │
│  - API definitions  │
└──────────┬──────────┘
           │ part of
           ▼
┌─────────────────────┐
│  kpt CLI (v1.x.x)   │
│  - Core tool        │
└─────────────────────┘
```

### Version Pinning **Recommended**: Pin SDK version in go.mod

```go
require (
    // Pin to specific version for reproducibility
    github.com/kptdev/krm-functions-sdk/go/fn v1.0.2
)
``` **Not Recommended**: Using version ranges

```go
require (
    // Avoid: may break with SDK updates
    github.com/kptdev/krm-functions-sdk/go/fn v1.0
)
```

### Updating Dependencies **Update SDK**:
```bash
# Update to latest v1.x.x
go get github.com/kptdev/krm-functions-sdk/go/fn@latest

# Update to specific version
go get github.com/kptdev/krm-functions-sdk/go/fn@v1.0.2

# Tidy dependencies
go mod tidy
``` **Test After Update**:
```bash
# Run function tests
go test ./...

# Test with kpt
kpt fn eval --image=my-function:dev test-package/
```

## Version Discovery

### Check SDK Version **In go.mod**:
```bash
grep krm-functions-sdk go.mod
``` **At Runtime**:
```go
import "github.com/kptdev/krm-functions-sdk/go/fn"

func main() {
    version := fn.SDKVersion()
    fmt.Printf("SDK Version: %s\n", version)
}
```

### Check Function Version **From Container**:
```bash
# Inspect container labels
docker inspect ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1 \
  | jq '.[0].Config.Labels'
``` **From Function Output**:
```bash
# Run function with --version flag (if supported)
docker run ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1 --version
```

## Best Practices

### For SDK Developers

1. **Follow Semver**: Strictly follow semantic versioning
2. **Document Changes**: Maintain detailed changelog
3. **Test Compatibility**: Test with multiple kpt versions
4. **Deprecate Gracefully**: Provide migration guides
5. **Version Metadata**: Include version in SDK code

### For Function Developers

1. **Pin SDK Version**: Use specific SDK version in go.mod
2. **Version Metadata**: Include version labels in container
3. **Test Thoroughly**: Test with target kpt versions
4. **Document Requirements**: Specify minimum kpt/SDK versions
5. **Follow Semver**: Version functions semantically

### For Function Users

1. **Pin Function Versions**: Use specific versions in Kptfile
2. **Test Updates**: Test function updates before production
3. **Check Compatibility**: Verify kpt/SDK compatibility
4. **Read Release Notes**: Review changes before updating
5. **Report Issues**: Report compatibility problems

## Examples

### Function with Version Metadata **Kptfile**:
```yaml
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: my-package
pipeline:
  mutators:
    - image: ghcr.io/kptdev/krm-functions-catalog/set-namespace:v0.4.1
      configMap:
        namespace: production
    - image: ghcr.io/kptdev/krm-functions-catalog/set-labels:v0.1.5
      configMap:
        app: myapp
```

### Function Development **go.mod**:
```go
module github.com/example/my-function

go 1.21

require (
    github.com/kptdev/krm-functions-sdk/go/fn v1.0.2
)
``` **main.go**:
```go
package main

import (
    "fmt"
    "os"
    
    "github.com/kptdev/krm-functions-sdk/go/fn"
)

const (
    Version    = "v1.0.0"
    SDKVersion = "v1.0.2"
    MinKptVersion = "v1.0.0"
)

func main() {
    if len(os.Args) > 1 && os.Args[1] == "--version" {
        fmt.Printf("Function Version: %s\n", Version)
        fmt.Printf("SDK Version: %s\n", SDKVersion)
        fmt.Printf("Min kpt Version: %s\n", MinKptVersion)
        return
    }
    
    if err := fn.AsMain(fn.ResourceListProcessorFunc(process)); err != nil {
        os.Exit(1)
    }
}

func process(rl *fn.ResourceList) (bool, error) {
    // Function logic
    return true, nil
}
```

## Troubleshooting

### SDK Version Mismatch **Error**:
```
Error: Function requires SDK v1.0.2 but uses v0.x.x
``` **Solution**:
```bash
go get github.com/kptdev/krm-functions-sdk/go/fn@v1.0.2
go mod tidy
```

### Function Compatibility Issue **Error**:
```
Error: Function set-namespace:v0.4.1 incompatible with kpt v1.0.0
``` **Solution**:
1. Check function documentation for kpt requirements
2. Update function to compatible version
3. Or update kpt to compatible version

### Version Detection Failed **Error**:
```
Warning: Cannot determine function version
``` **Solution**:
Add version metadata to function container:
```dockerfile
LABEL version="v1.0.0"
LABEL sdk-version="v1.0.2"
```

## References

- [kpt Versioning Policy](./VERSIONING.md)
- [Backward Compatibility](./BACKWARD_COMPATIBILITY.md)
- [SDK Repository](https://github.com/kptdev/krm-functions-sdk)
- [Function Catalog](https://github.com/kptdev/krm-functions-catalog)
- [Function Catalog Website](https://catalog.kpt.dev/)
- [Semantic Versioning](https://semver.org/)
