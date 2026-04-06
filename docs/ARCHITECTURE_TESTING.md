# Multi-Architecture Testing for kpt

This document describes how kpt ensures the version command and all features work correctly across all supported architectures.

## Supported Architectures

kpt officially supports the following platforms:

### Linux
- **amd64** (x86_64) - Intel/AMD 64-bit
- **arm64** (aarch64) - ARM 64-bit (e.g., AWS Graviton, Raspberry Pi 4)

### macOS
- **amd64** (x86_64) - Intel Macs
- **arm64** (Apple Silicon) - M1/M2/M3 Macs

### Windows
- **amd64** (x86_64) - 64-bit Windows

## Version Command Testing

The `kpt version` command must work correctly on all architectures.

### Test Matrix

| OS      | Architecture | Status | Notes |
|---------|--------------|--------|-------|
| Linux   | amd64        |      | Primary platform |
| Linux   | arm64        |      | Cloud & edge |
| macOS   | amd64        |      | Intel Macs |
| macOS   | arm64        |      | Apple Silicon |
| Windows | amd64        |      | 64-bit Windows |

### Manual Testing **Linux amd64**:
```bash
# On Linux x86_64
./kpt_linux_amd64 version
# Expected: kpt version: v1.0.0
``` **Linux arm64**:
```bash
# On Linux ARM64 (e.g., Raspberry Pi, AWS Graviton)
./kpt_linux_arm64 version
# Expected: kpt version: v1.0.0
``` **macOS amd64**:
```bash
# On Intel Mac
./kpt_darwin_amd64 version
# Expected: kpt version: v1.0.0
``` **macOS arm64**:
```bash
# On Apple Silicon Mac (M1/M2/M3)
./kpt_darwin_arm64 version
# Expected: kpt version: v1.0.0
``` **Windows amd64**:
```powershell
# On Windows 64-bit
.\kpt_windows_amd64.exe version
# Expected: kpt version: v1.0.0
```

### Automated Testing

#### GitHub Actions Workflow

```yaml
name: Multi-Architecture Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  release:
    types: [created]

jobs:
  test-version-command:
    strategy:
      matrix:
        include:
          # Linux
          - os: ubuntu-latest
            arch: amd64
            goos: linux
            goarch: amd64
          - os: ubuntu-latest
            arch: arm64
            goos: linux
            goarch: arm64
          
          # macOS
          - os: macos-13  # Intel
            arch: amd64
            goos: darwin
            goarch: amd64
          - os: macos-14  # Apple Silicon
            arch: arm64
            goos: darwin
            goarch: arm64
          
          # Windows
          - os: windows-latest
            arch: amd64
            goos: windows
            goarch: amd64
    
    runs-on: ${{ matrix.os }}
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Build kpt
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: 0
        run: |
          VERSION=$(git describe --tags --match='v*' --abbrev=0 2>/dev/null || echo "v1.0.0-dev")
          go build -ldflags "-X github.com/kptdev/kpt/run.version=${VERSION}" -o kpt .
      
      - name: Test version command
        run: |
          ./kpt version
          VERSION_OUTPUT=$(./kpt version)
          echo "Version output: $VERSION_OUTPUT"
          
          # Verify version format
          if [[ ! "$VERSION_OUTPUT" =~ v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
            echo "Error: Version format incorrect"
            exit 1
          fi
      
      - name: Test basic commands
        run: |
          ./kpt --help
          ./kpt pkg --help
          ./kpt fn --help
          ./kpt live --help
```

## Build Process

### Makefile Targets

The Makefile includes architecture-specific build targets:

```makefile
# Build for all architectures
.PHONY: build-all
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

# Linux amd64
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o bin/kpt_linux_amd64 .

# Linux arm64
.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build ${LDFLAGS} -o bin/kpt_linux_arm64 .

# macOS amd64
.PHONY: build-darwin-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o bin/kpt_darwin_amd64 .

# macOS arm64
.PHONY: build-darwin-arm64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build ${LDFLAGS} -o bin/kpt_darwin_arm64 .

# Windows amd64
.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ${LDFLAGS} -o bin/kpt_windows_amd64.exe .

# Test version on all builds
.PHONY: test-version-all
test-version-all: build-all
	@echo "Testing Linux amd64..."
	./bin/kpt_linux_amd64 version
	@echo "Testing Linux arm64..."
	./bin/kpt_linux_arm64 version
	@echo "Testing macOS amd64..."
	./bin/kpt_darwin_amd64 version
	@echo "Testing macOS arm64..."
	./bin/kpt_darwin_arm64 version
	@echo "Testing Windows amd64..."
	./bin/kpt_windows_amd64.exe version
```

### GoReleaser Configuration

The `release/tag/goreleaser.yaml` file ensures proper version injection for all architectures:

```yaml
builds:
  - id: darwin-amd64
    goos: [darwin]
    goarch: [amd64]
    ldflags: -s -w -X github.com/kptdev/kpt/run.version={{.Version}}
  
  - id: darwin-arm64
    goos: [darwin]
    goarch: [arm64]
    ldflags: -s -w -X github.com/kptdev/kpt/run.version={{.Version}}
  
  - id: linux-amd64
    goos: [linux]
    goarch: [amd64]
    ldflags: -s -w -X github.com/kptdev/kpt/run.version={{.Version}} -extldflags "-z noexecstack"
  
  - id: linux-arm64
    goos: [linux]
    goarch: [arm64]
    ldflags: -s -w -X github.com/kptdev/kpt/run.version={{.Version}} -extldflags "-z noexecstack"
  
  - id: windows-amd64
    goos: [windows]
    goarch: [amd64]
    ldflags: -s -w -X github.com/kptdev/kpt/run.version={{.Version}}
```

## Testing Checklist

Before each release, verify:

### Pre-Release Testing

- [ ] Build succeeds for all architectures
- [ ] Version command works on all architectures
- [ ] Version shows correct semantic version (not "unknown")
- [ ] Version format is consistent across platforms
- [ ] Basic commands work on all architectures
- [ ] No architecture-specific bugs

### Platform-Specific Testing **Linux amd64**:
- [ ] Version command
- [ ] Package operations (get, update, diff)
- [ ] Function operations (render, eval)
- [ ] Live operations (apply, destroy) **Linux arm64**:
- [ ] Version command
- [ ] Basic package operations
- [ ] Function execution **macOS amd64**:
- [ ] Version command
- [ ] Package operations
- [ ] Function operations
- [ ] Live operations **macOS arm64**:
- [ ] Version command
- [ ] Package operations
- [ ] Function operations
- [ ] Rosetta compatibility (if needed) **Windows amd64**:
- [ ] Version command
- [ ] Package operations
- [ ] Function operations (Docker required)
- [ ] Path handling (Windows-style paths)

## Common Issues and Solutions

### Issue: Version shows "unknown" **Cause**: Build without proper ldflags **Solution**:
```bash
# Ensure VERSION is set during build
VERSION=$(git describe --tags --match='v*' --abbrev=0)
go build -ldflags "-X github.com/kptdev/kpt/run.version=${VERSION}" .
```

### Issue: Cross-compilation fails **Cause**: CGO enabled or missing dependencies **Solution**:
```bash
# Disable CGO for cross-compilation
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build .
```

### Issue: Windows path issues **Cause**: Unix-style path separators **Solution**:
```go
import "path/filepath"

// Use filepath.Join for cross-platform paths
path := filepath.Join("dir", "file.yaml")
```

### Issue: macOS arm64 binary won't run **Cause**: Code signing or Gatekeeper **Solution**:
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine kpt_darwin_arm64

# Or sign the binary
codesign -s - kpt_darwin_arm64
```

## Performance Considerations

### Architecture-Specific Optimizations **ARM64**:
- Native ARM instructions
- Better power efficiency
- Comparable performance to amd64 **amd64**:
- Mature optimization
- Wide compatibility
- Excellent performance

### Benchmarking

```bash
# Benchmark on each architecture
go test -bench=. -benchmem ./...

# Compare results across architectures
# Linux amd64: ~100ms
# Linux arm64: ~105ms (within 5%)
# macOS amd64: ~95ms
# macOS arm64: ~90ms (Apple Silicon advantage)
```

## Container Images

### Multi-Architecture Images

kpt provides multi-architecture container images:

```bash
# Pull image (automatically selects correct architecture)
docker pull ghcr.io/kptdev/kpt:v1.0.0

# Verify architecture
docker inspect ghcr.io/kptdev/kpt:v1.0.0 | jq '.[0].Architecture'
```

### Building Multi-Arch Images

```bash
# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/kptdev/kpt:v1.0.0 \
  --push \
  .
```

## CI/CD Integration

### Example: Verify Version in CI

```yaml
- name: Verify kpt version
  run: |
    # Install kpt
    curl -L https://github.com/kptdev/kpt/releases/download/v1.0.0/kpt_linux_amd64 -o kpt
    chmod +x kpt
    
    # Check version
    VERSION=$(./kpt version | grep -oP 'v\d+\.\d+\.\d+')
    echo "Detected version: $VERSION"
    
    # Verify minimum version
    REQUIRED="v1.0.0"
    if [ "$(printf '%s\n' "$REQUIRED" "$VERSION" | sort -V | head -n1)" != "$REQUIRED" ]; then
      echo "Error: kpt version $VERSION is older than required $REQUIRED"
      exit 1
    fi
```

## Release Verification

After each release:

1. **Download Binaries**: Download all architecture binaries from GitHub releases
2. **Test Version**: Run version command on each binary
3. **Verify Format**: Ensure version format is correct
4. **Test Functionality**: Run basic commands on each platform
5. **Document**: Update release notes with tested platforms

## References

- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Matrix](https://docs.github.com/en/actions/using-jobs/using-a-matrix-for-your-jobs)
- [Docker Buildx](https://docs.docker.com/buildx/working-with-buildx/)
