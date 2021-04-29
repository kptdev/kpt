A package can be _composed_ of subpackages (_HAS A_ relationship). _Package composition_ is when you
change the package hierarchy by adding or removing subpackages.

There are two different ways to add a subpackage to a package on the local filesystem:

1. [Create a new package] in a subdirectory
2. [Get an existing package] in a subdirectory

Let's revisit the `wordpress` package and see how it was composed in the first place.
Currently, it has the following package hierarchy:

```yaml
$ kpt pkg tree wordpress/
PKG: wordpress
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── PKG: mysql
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [deployment.yaml]  Service wordpress-mysql
```

First, let's delete the `mysql` subpackage. Deleting a subpackage is done by simply deleting the
subdirectory:

```shell
rm -r wordpress/mysql
```

We're going to add back the `mysql` subpackage using the two different approaches:

### Create a new package

```shell
$ cd wordpress
$ mkdir mysql
$ kpt pkg init mysql
# author resources in mysql
```

This creates a [dependent package].

### Get an existing package

```shell
$ cd wordpress
$ kpt pkg get https://github.com/kubernetes/website.git/content/en/examples/application/mysql@snapshot-initial-v1.20
```

TODO(#1829): This can be simplified.

This creates a [independent package]. If you wish this to make this a dependent package, you
can delete the `upstream` and `upstreamLock` sections of the `Kptfile` in `mysql` directory.

[create a new package]: /book/03-packages/05-creating-a-package
[get an existing package]: /book/03-packages/01-getting-a-package
[dependent package]: /book/03-packages/01-getting-a-package
[independent package]: /book/03-packages/01-getting-a-package
