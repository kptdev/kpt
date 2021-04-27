---
title: "Update"
linkTitle: "update"
type: docs
description: >
   Apply upstream package updates.
---
<!--mdtogo:Short
    Apply upstream package updates.
-->

`update` pulls in upstream changes and merges them into a local package.
Changes may be applied using one of several strategies.

Since this will update the local package, all changes must be committed to 
git before running `update`

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg update [PKG_PATH][@VERSION] [flags]
```

#### Args

```
PKG_PATH:
  Local package path to update. Directory must exist and contain a Kptfile
  to be updated. Defaults to the current working directory.

VERSION:
  A git tag, branch, ref or commit. Specified after the local_package
  with @ -- pkg@version.
  Defaults the ref specified in the Upstream section of the package Kptfile.

  Version types:
    * branch: update the local contents to the tip of the remote branch
    * tag: update the local contents to the remote tag
    * commit: update the local contents to the remote commit
```

#### Flags

```
--strategy:
  Defines which strategy should be used to update the package. This will change
  the update strategy for the current kpt package for the current and future 
  updates. If a strategy is not provided, the strategy specified in the package
  Kptfile will be used.

    * resource-merge: Perform a structural comparison of the original /
      updated resources, and merge the changes into the local package.
    * fast-forward: Fail without updating if the local package was modified
      since it was fetched.
    * force-delete-replace: Wipe all the local changes to the package and replace
      it with the remote version.
```

#### Env Vars

```
KPT_CACHE_DIR:
  Controls where to cache remote packages when fetching them.
  Defaults to <HOME>/.kpt/repos/
  On macOS and Linux <HOME> is determined by the $HOME env variable, while on
  Windows it is given by the %USERPROFILE% env variable.
```
<!--mdtogo-->

### Examples
<!--mdtogo:Examples-->
```shell
# Update package in the current directory.
git add . && git commit -m 'some message'
kpt pkg update
```

```shell
# Update my-package-dir/ to match the v1.3 branch or tag.
git add . && git commit -m 'some message'
kpt pkg update my-package-dir/@v1.3
```

```shell
# Update with the fast-forward strategy.
git add . && git commit -m "package updates"
kpt pkg update my-package-dir/@master --strategy fast-forward
```
<!--mdtogo-->
