---
title: "`tree`"
linkTitle: "tree"
weight: 4

description: |
  Display resources, files and packages in a tree structure.
---

<!--mdtogo:Short
    Display resources, files and packages in a tree structure.
-->

`tree` displays resources, files and packages in a tree structure.

Each package (directory containing a `Kptfile`) is annotated with its type:

- **(independent)**: The package has an `upstream` section in its `Kptfile`,
  meaning it tracks its own upstream source and can be updated independently.
- **(dependent)**: A subpackage that does not have an `upstream` section in its
  `Kptfile`. Its upstream is implicitly inherited from its parent package.

A root package created locally (e.g. via `kpt pkg init`) has no annotation since
it is neither independent nor dependent — it simply has no upstream source.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt pkg tree [DIR]
```

#### Args

```shell
DIR:
  Path to a local package directory. Defaults to the current directory.
  Displays KRM resources with their Kind and Name, and non-KRM text files
  as plain filenames. Dotfiles and symlinks are excluded.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Show resources in the current directory.
$ kpt pkg tree
```

```shell
# Example output showing independent and dependent packages:
$ kpt pkg tree wordpress/
Package "wordpress" (independent)
├── [Kptfile]  Kptfile wordpress
├── [service.yaml]  Service wordpress
├── deployment
│   ├── [deployment.yaml]  Deployment wordpress
│   └── [volume.yaml]  PersistentVolumeClaim wp-pv-claim
└── Package "mysql" (dependent)
    ├── [Kptfile]  Kptfile mysql
    ├── [deployment.yaml]  Deployment wordpress-mysql
    └── [service.yaml]  Service wordpress-mysql
```

<!--mdtogo-->
