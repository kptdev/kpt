A kpt package is a bundle of configuration _data_. It is represented as a
directory tree containing KRM resources using YAML as the file format.

A package is explicitly declared using a file named `Kptfile` containing a KRM
resource of kind `Kptfile`. The Kptfile contains metadata about the package and
is just a regular resource in the YAML format.

Just as directories can be nested, a package can contain another package, called
a _subpackage_.

Let's take a look at the wordpress package as an example:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@v0.9
```

View the package hierarchy using the `tree` command:

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

This _package hierarchy_ contains two packages:

1. `wordpress` is the top-level package in the hierarchy declared using
   `wordpress/Kptfile`. This package contains 2 subdirectories.
   `wordpress/deployment` is a regular directory used for organizing resources
   that belong to the `wordpress` package itself. The `wordpress` package
   contains 3 direct resources in 3 files: `service.yaml`,
   `deployment/deployment.yaml`, and `deployment/volume.yaml`.
2. `wordpress/mysql` is a subpackage of `wordpress` package since it contains a
   `Kptfile`. This package contains 3 resources in
   `wordpress/mysql/deployment.yaml` file.

kpt uses Git as the underlying version control system. A typical workflow starts
by fetching an _upstream_ package from a Git repository to the local filesystem
using `kpt pkg` commands. All other functionality (i.e. `kpt fn` and `kpt live`)
use the package from the local filesystem, not the remote Git repository. You
may think of this as the _vendoring_ used by tooling for some programming
languages. The main difference is that kpt is designed to enable you to modify
the vendored package on the local filesystem and then later update the package
by merging the local and upstream changes.

There is one scenario where a Kptfile is implicit: You can use kpt to fetch any
Git directory containing KRM resources, even if it does not contain a `Kptfile`.
Effectively, you are telling kpt to treat that Git directory as a package. kpt
automatically creates the `Kptfile` on the local filesystem to keep track of the
upstream repo. This means that kpt is compatible with large corpus of existing
Kubernetes configuration stored on Git today!

For example, `cockroachdb` is just a vanilla directory of KRM:

```shell
$ kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb
```

We will go into details of how to work with packages in [Chapter 3].

[chapter 3]: /book/03-packages/
