## Description of the change

Modified `kpt pkg init` to automatically create the package directory if it doesn't exist, instead of throwing an error.

## Motivation for the change

Currently, users must manually create a directory before running `kpt pkg init`. This adds an unnecessary step to the workflow. With this change, the command creates the directory automatically, making package initialization more convenient.

**Before:**
```bash
mkdir my-package
kpt pkg init my-package
```

**After:**
```bash
kpt pkg init my-package  # Directory created automatically
```

## What issue it fixes

Fixes #1835

## Additional context

- Supports nested paths: `kpt pkg init path/to/package`
- Backward compatible: existing directories continue to work
- All tests passing
