# prod

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] prod`
Details: https://kpt.dev/reference/pkg/get/

### View package content
`kpt pkg tree prod`
Details: https://kpt.dev/reference/pkg/tree/

### Apply the package
```
kpt live init prod
kpt live apply prod --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/live/
