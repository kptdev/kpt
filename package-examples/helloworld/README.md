# helloworld

Sample helloworld package. Showcases what a bare bones package is which
doesn't do anything beyond declaring the current directory as a `kpt` package.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`

{{% hide %}}

<!-- @makeWorkplace @verifyPkgExamples-->
```
# Set up workspace for the test.
setupWorkspace

# Create output file.
createOutputFile
```
<!-- @pkgGet @verifyPkgExamples-->
```shell
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld
```

{{% /hide %}}

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld@next

fetching package /package-examples/helloworld from https://github.com/GoogleContainerTools/kpt to helloworld
```

### View the package contents

List the package contents in a tree structure.


{{% hide %}}

<!-- @pkgTree @verifyPkgExamples-->
```shell
kpt pkg tree helloworld/ > output.txt
expectedOutput "Package \"helloworld\"
├── [Kptfile]  Kptfile helloworld
├── [deploy.yaml]  Deployment helloworld-gke
└── [service.yaml]  Service helloworld-gke"
```

{{% /hide %}}

```shell
$ kpt pkg tree helloworld/

Package "helloworld"
├── [Kptfile]  Kptfile helloworld
├── [deploy.yaml]  Deployment helloworld-gke
└── [service.yaml]  Service helloworld-gke
```

### Apply the package

Initialize the inventory object:

```shell
$ kpt live init helloworld/
```

Apply all the contents of the package recursively to the cluster

```shell
$ kpt live apply helloworld/

service/helloworld-gke created
deployment.apps/helloworld-gke created
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```