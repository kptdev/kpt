After you fetch a package to your local filesystem, you typically want to
explore the package to understand how it is composed and how it can be
customized for your needs. Given a kpt package is just an ordinary directory of
human-readable YAML files, you can naturally use your favorite file explorer,
shell commands, or editor to explore the package.

kpt also provides the `tree` command which is handy for quickly viewing package
hierarchy and the constituent packages, files, and resources:

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

?> Refer to the [tree command reference][tree-doc] for usage.

In addition, you can use a kpt function such as `search-replace` to run a query
on the package. For example, to search for resources that have a field with path
`spec.selector.tier`:

```shell
$ kpt fn eval wordpress -i search-replace:v0.1 -- 'by-path=spec.selector.tier'
```

[tree-doc]: /reference/cli/pkg/tree/
