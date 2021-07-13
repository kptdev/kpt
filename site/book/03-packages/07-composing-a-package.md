A package can be _composed_ of subpackages (_HAS A_ relationship). _Package
composition_ is when you change the package hierarchy by adding or removing
subpackages.

There are two different ways to add a subpackage to a package on the local
filesystem:

1. [Create a new package] in a subdirectory
2. [Get an existing package] in a subdirectory

Let's revisit the `wordpress` package and see how it was composed in the first
place. Currently, it has the following package hierarchy:

```shell
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

First, let's delete the `mysql` subpackage. Deleting a subpackage is done by
simply deleting the subdirectory:

```shell
$ rm -r wordpress/mysql
```

We're going to add back the `mysql` subpackage using the two different
approaches:

## Create a new package

Create the directory:

```shell
$ mkdir wordpress/mysql
```

Initialize the package:

```shell
$ kpt pkg init wordpress/mysql
# author resources in mysql
```

This creates a [dependent package].

## Get an existing package

Remove the existing directory if it exists:

```shell
$ rm -rf wordpress/mysql
```

Fetch the package:

```shell
$ kpt pkg get https://github.com/kubernetes/website.git/content/en/examples/application/mysql@snapshot-initial-v1.20 wordpress/mysql
```

This creates an [independent package]. If you wish to make this a dependent
package, you can delete the `upstream` and `upstreamLock` sections of the
`Kptfile` in `mysql` directory.

[create a new package]: /book/03-packages/06-creating-a-package
[get an existing package]: /book/03-packages/01-getting-a-package
[dependent package]: /book/03-packages/01-getting-a-package
[independent package]: /book/03-packages/01-getting-a-package
