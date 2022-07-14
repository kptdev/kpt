# cert-manager-basic

## Description
sample description

## Usage

### Fetch the package
`kpt pkg get REPO_URI[.git]/PKG_PATH[@VERSION] cert-manager-basic`
Details: https://kpt.dev/reference/cli/pkg/get/

### View package content
`kpt pkg tree cert-manager-basic`
Details: https://kpt.dev/reference/cli/pkg/tree/

### Apply the package
```
kpt live init cert-manager-basic
kpt live apply cert-manager-basic --reconcile-timeout=2m --output=table
```
Details: https://kpt.dev/reference/cli/live/
