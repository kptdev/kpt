# helloworld

Sample helloworld package.  Showcases what a bare bones package is which 
doesn't do anything beyond declaring the current directory as a `kpt` package. 

## Steps

This is a simple workflow on a package that requires no configuration or
customization.

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

  $ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld

    fetching package /package-examples/helloworld from https://github.com/GoogleContainerTools/kpt to helloworld

### View the package contents

List the package contents in a tree structure.

  $ kpt pkg tree helloworld/

    PKG: helloworld
    ├── [Kptfile]  Kptfile helloworld
    ├── [deploy.yaml]  Deployment helloworld-gke
    └── [service.yaml]  Service helloworld-gke

### Apply the package

Initialize the inventory object:

  $ kpt live init helloworld

    namespace: default is used for inventory object
    Initialized: helloworld/inventory-template.yaml


Apply all the contents of the package recursively to the cluster

  $ kpt live apply helloworld 

    service/helloworld-gke created
    deployment.apps/helloworld-gke created
    2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
    0 resource(s) pruned, 0 skipped, 0 failed

[tree]: ../../../site/reference/pkg/tree
[live init]: ../../../site/reference/live/init
[live apply]: ../../../site/reference/live/apply
