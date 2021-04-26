# helloworld-tshirt

This is a package that has both a setter and a simple hydration function.  The
hydration function examines the configuration and looks for the t-shirt size
value.  Depending on the size (small, medium, large) it then sets the 
resource sizes.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

```
  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-tshirt

    fetching package /package-examples/helloworld-tshirt from https://github.com/GoogleContainerTools/kpt to helloworld-tshirt
```

### View the package contents

List the package contents in a tree structure.

```
  $ kpt pkg tree helloworld-tshirt/

    PKG: helloworld-tshirt
    ├── [Kptfile]  Kptfile helloworld-tshirt
    ├── [helloworld.yaml]  Deployment helloworld-gke
    └── [helloworld.yaml]  Service helloworld-gke
```

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `apply-setters` and 'example-tshirt' functions.  The `apply-setters` 
function allows you to set a simple value throughout the package 
configuration.  In this case the value sets the t-shirt size.  You can 
set it to medium.

```
  pipeline:
    mutators:
      - image: gcr.io/kpt-fn/apply-setters:unstable
        configMap:
          tshirt-size: medium
      - image: gcr.io/kustomize-functions/example-tshirt:v0.1.0
```

### Render the declared values

Render the changes in the hydration pipeline by using `kpt fn render` command:

```
  $ kpt fn render helloworld-tshirt/

    package "helloworld-tshirt": running function "gcr.io/kpt-fn/apply-setters:unstable": SUCCESS
    package "helloworld-tshirt": running function "gcr.io/kustomize-functions/example-tshirt:v0.1.0": SUCCESS
    package "helloworld-tshirt": rendered successfully
```

### Apply the package

Initialize the inventory object:

```
  $ kpt live init helloworld-tshirt/
```

Apply all the contents of the package recursively to the cluster

```
  $ kpt live apply helloworld-tshirt/

    service/helloworld-gke created
    deployment.apps/helloworld-gke created
    2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
    0 resource(s) pruned, 0 skipped, 0 failed
```
