# wordpress

Here is an example to get, view, customize and apply contents of an example kpt package with a subpackage
in its directory tree. As part of customization, we will be setting common namespace for both the packages
and apply setters for individual packages.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [Configure namespace](#configure-namespace)
4. [Configure setter values](#configure-setter-values)
5. [Render the declared values](#render-the-declared-values)
6. [Apply the package](#apply-the-package)

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
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@next

```

{{% /hide %}}

```sh
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@next

fetching package /package-examples/wordpress from https://github.com/GoogleContainerTools/kpt to wordpress
```

### View the package contents

List the package contents in a tree structure.

{{% hide %}}

<!-- @pkgTree @verifyPkgExamples-->
```shell
kpt pkg tree wordpress/ > output.txt
expectedOutput "Package \"wordpress\"
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Package \"mysql\"
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [deployment.yaml]  Service wordpress-mysql"
```

{{% /hide %}}

```sh
$ kpt pkg tree wordpress/

Package "wordpress"
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Package "mysql"
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [deployment.yaml]  Service wordpress-mysql
```

### Configure namespace

By default, these packages will be deployed into `default` namespace. Provide a namespace by
adding [set-namespace] function to the pipeline definition in `wordpress/Kptfile`.

```yaml
- image: gcr.io/kpt-fn/set-namespace:v0.1
  configMap:
    namespace: my-space
```

### Configure setter values

Setters are listed under `apply-setters` function in the pipeline definition of each package.
You may declare new desired values for the setters by editing the `Kptfile` directly. Declare
new value `wp-tag: 4.9-aapache` in `wordpress/Kptfile` and `ms-tag: 5.7` in `wordpress/mysql/Kptfile`

### Render the declared values

```sh
$ kpt fn render wordpress/
```

### Apply the package

Apply all the contents of the package recursively to the cluster

```sh
$ kpt live init wordpress/

namespace: default is used for inventory object
Initialized: wordpress/inventory-template.yaml
```

```sh
$ kubectl create ns my-space

namespace/my-space created
```

```sh
$ kpt live apply wordpress/

service/wordpress-mysql created
persistentvolumeclaim/mysql-pv-claim created
deployment.apps/wordpress-mysql created
service/wordpress created
persistentvolumeclaim/wp-pv-claim created
deployment.apps/wordpress created
```

[set-namespace]: https://github.com/GoogleContainerTools/kpt-functions-catalog/tree/master/functions/go/set-namespace
