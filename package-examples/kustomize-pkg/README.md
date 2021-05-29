# package-name

This is a simple package that shows how kpt can be used to replace
remote bases in kustomize.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/kustomize-pkg

fetching package /package-examples/kustomize-pkg from https://github.com/GoogleContainerTools/kpt to kustomize-pkg
```

### View the package contents

List the package contents in a tree structure.

```shell
$ kpt pkg tree kustomize-pkg/


```

### Run kustomize build

Since this package is using 

### Render the declared values

This step should describe the process and exact steps to render the declared values.

### Apply the package

This step should describe the process and exact commands needed to deploy the resources to a live cluster.
