---
title: "`plan`"
linkTitle: "plan"
type: docs
description: >
  Output a plan for the changes that will happen when applying a package.
---

<!--mdtogo:Short
    Output a plan for the changes that will happen when applying a package.
-->

?> This feature is still in alpha, so the UX and the output format is subject to change.

`plan` does a dry-run of applying a package to the cluster. It outputs the results
in combination with a diff for every resource that will be updated, which gives an
overview of the impact of applying a package.

Note that `plan` does only works reliably with server-side apply.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha live plan [PKG_PATH | -] [flags]
```

#### Args

```
PKG_PATH | -:
  Path to the local package which should be applied to the cluster. It must
  contain a Kptfile with inventory information. Defaults to the current working
  directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Flags

```
--field-manager:
  Identifier for the **owner** of the fields being applied. Only usable
  when --server-side flag is specified. Default value is kubectl.

--force-conflicts:
  Force overwrite of field conflicts during apply due to different field
  managers. Only usable when --server-side flag is specified.
  Default value is false (error and failure when field managers conflict).

--inventory-policy:
  Determines how to handle overlaps between the package being currently applied
  and existing resources in the cluster. The available options are:

    * strict: If any of the resources already exist in the cluster, but doesn't
      belong to the current package, it is considered an error.
    * adopt: If a resource already exist in the cluster, but belongs to a
      different package, it is considered an error. Resources that doesn't belong
      to other packages are adopted into the current package.

  The default value is `strict`.

--output:
  Determines the output format for the plan. Must be one of the following:

    * text: The plan will be printed as text to stdout.
    * krm: The plan will be printed as a Plan KRM resource to stdout. This
      can be used as input to kpt functions for automatic validation.

  The default value is ‘text’.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# create a plan for the package in the current directory and output in KRM format.
$ kpt alpha live plan --output=krm
```
<!--mdtogo-->
