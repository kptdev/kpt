# dev

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] dev`
Details: https://kpt.dev/reference/pkg/get/

### View package content
`kpt pkg tree dev`
Details: https://kpt.dev/reference/pkg/tree/

### Apply the package
```
kpt live init dev
kpt live apply dev --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/live/
