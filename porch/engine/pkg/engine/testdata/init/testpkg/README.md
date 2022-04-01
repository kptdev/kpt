# testpkg

## Description
test package

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] testpkg`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree testpkg`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init testpkg
kpt live apply testpkg --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
