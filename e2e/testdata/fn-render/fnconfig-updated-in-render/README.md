# app-example

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] app-example`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree app-example`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init app-example
kpt live apply app-example --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
