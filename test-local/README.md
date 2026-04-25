# test-local

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] test-local`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree test-local`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init test-local
kpt live apply test-local --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
