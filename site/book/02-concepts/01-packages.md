A kpt package is a bundle of configuration _data_. It is represented as a directory tree containing
KRM resources using YAML as the file format.

A package is explicitly declared using a file named `Kptfile` containing a KRM resource of kind
`Kptfile`. The Kptfile contains metadata about the package and is just a regular resource in the YAML format.

Just as directories can be nested, a package can contain another package, called a
_subpackage_.

Let's take a look at an example:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress@next \
wordpress
$ kpt pkg tree wordpress/
wordpress
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [service.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Pkg: mysql
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  PersistentVolumeClaim mysql-pv-claim
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [deployment.yaml]  Service wordpress-mysql
```

There are two packages in this example. The top-level `wordpress` package is declared using
`wordpress/Kptfile`. This package contains 2 subdirectories. `wordpress/deployment` is a regular
directory used for organizing resources that belong to the `wordpress` package itself. On the other hand,
`wordpress/mysql` is a subpackage since it has a `Kptfile`. The `mysql` package contains 3
resources in its `deployment.yaml` file.

The `Kptfile` is an example of a _meta resource_ in kpt. Meta resources are resources that
are only needed for kpt itself to function, they do not have extrinsic meaning and are not
applied to a cluster. We will see another example of a meta resources in the next section.

kpt uses Git as the underlying version control system. A typical workflow starts by fetching an
_upstream_ package from a Git to the local filesystem using `kpt pkg` commands. All other
functionality (i.e. `kpt fn` and `kpt live`) use the package from the local filesystem, not the
remote Git repository. You may think of this as the _vendoring_ concept from some programming
languages. The main difference is that kpt is designed to enable you to modify the vendored package
on the local filesystem and then later update the package by merging the local and upstream changes.

There is one scenario where a Kptfile is implicit: You can use kpt to fetch any Git directory
containing KRM resources, even if it does not contain a `Kptfile`. Effectively, you are telling kpt
to treat that Git directory as a package. kpt automatically creates the `Kptfile`
on the local filesystem to keep track of the upstream repo. This means that kpt is
compatible with large corpus of existing Kubernetes configuration stored on Git today!

For example, `cockroachdb` is just a vanilla directory of KRM:

```shell
kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb cockroachdb
```

We will go into details of how to work with packages in Chapter 3.
