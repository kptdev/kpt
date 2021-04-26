# helloworld-kustomize

This is an example of a kpt package that has a kustomize
patch in it.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

```
    $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-kustomize

      fetching package /package-examples/helloworld-kustomize from https://github.com/GoogleContainerTools/kpt to helloworld-kustomize
```

### View the package contents

List the package contents in a tree structure.

```
    $ kpt pkg tree helloworld-kustomize/

      PKG: helloworld-kustomize
      ├── [Kptfile]  Kptfile helloworld-kustomize
      ├── patches
      │   └── [patch.yaml]  Deployment helloworld-gke
      └── resources
          ├── [deploy.yaml]  Deployment helloworld-gke
          └── [service.yaml]  Service helloworld-gke
```

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `apply-setters` function.  The `apply-setters` function allows you to
set a simple value throughout the package configuration.  In this case the
value of the setter goes into the `patch.yaml`.  You can set the target
environment variable to a value different of your choice (different
than `foobar`)

```
    pipeline:
      mutators:
        - image: gcr.io/kpt-fn/apply-setters:unstable
          configMap:
            target: foobar
```

### Render the declared values

Render the changes in the hydration pipeline by using `kpt fn render` command:

```
    $ kpt fn render helloworld-kustomize/
```

### Apply the package

Since this is a kustomize example we will be using `kubectl -k`:

```
    $ kubectl apply -k helloworld-kustomize

      service/helloworld-gke created
      deployment.apps/helloworld-gke created
```