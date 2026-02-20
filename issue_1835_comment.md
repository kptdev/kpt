## Solution

Changed `kpt pkg init` to automatically create the package directory if it doesn't exist.

**Modified Files:**
- `pkg/kptpkg/init.go` - Replace error with `MkdirAll()` call
- `commands/pkg/init/cmdinit_test.go` - Updated test to verify auto-creation

**Before:**
```bash
$ kpt pkg init my-package
Error: my-package does not exist
```

**After:**
```bash
$ kpt pkg init my-package
creating package directory my-package
writing my-package/Kptfile
writing my-package/README.md
writing my-package/package-context.yaml
```

**Features:**
- Supports nested paths: `kpt pkg init path/to/package`
- Backward compatible with existing directories
- All tests passing

Fixes #1835
