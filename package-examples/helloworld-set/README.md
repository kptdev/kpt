# helloworld-set

This is a simple package that showcases a kpt package that has a number of
setters in it.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure functions](#configure-functions)
4. [Render the declared values](#render-the-declared-values)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set

fetching package /package-examples/helloworld-set from https://github.com/GoogleContainerTools/kpt to helloworld-set
```

### View the package contents

List the package contents in a tree structure.

```shell
$ kpt pkg tree helloworld-set/

Package "helloworld-set"
├── [Kptfile]  Kptfile helloworld-set
├── [deploy.yaml]  Deployment helloworld-gke
└── [service.yaml]  Service helloworld-gke
```

### Configure functions

The package contains a function pipeline in the `Kptfile` which has
one `apply-setters` function.  The `apply-setters` function allows you to
set a simple value throughout the package configuration.  In this case
you can set the replicas, image, tag and http-port of a simple application.

```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/apply-setters:unstable
      configMap:
        replicas: 5
        image: gcr.io/kpt-dev/helloworld-gke
        tag: latest
        http-port: 80
```

### Render the declared values

Render the changes in the hydration pipeline by using `kpt fn render` command:

```shell
$ kpt fn render helloworld-set/

package "helloworld-set": running function "gcr.io/kpt-fn/apply-setters:unstable": SUCCESS
package "helloworld-set": rendered successfully
```

### Apply the package

Initialize the inventory object:

```shell
$ kpt live init helloworld-set/
```

Apply all the contents of the package recursively to the cluster

```shell
$ kpt live apply helloworld-set/

service/helloworld-gke created
deployment.apps/helloworld-gke created
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```
