---
title: "`clone`"
linkTitle: "clone"
type: docs
description: >
Update a downstream package revision to a more recent revision of its upstream package.
---

<!--mdtogo:Short
    Update a downstream package revision to a more recent revision of its upstream package.
-->

`update` performs a kpt pkg update on an existing downstream package revision.

### Synopsis

<!--mdtogo:Long-->

```
kpt alpha rpkg update PACKAGE_REV_NAME [flags]
```

#### Args

```
PACKAGE_REV_NAME:
The target downstream package revision to be updated.

```

#### Flags

```
--revision
The revision number of the upstream kpt package that the target
downstream package (PACKAGE_REV_NAME) should be updated to. With
this flag, you can only specify one target downstream package.

--discover
If set, list packages revisions that need updates rather than
performing an update. Must be one of 'upstream' or 'downstream'. If
set to 'upstream', this will list downstream package revisions that
have upstream updates available. If set to 'downstream', this will list
upstream package revisions whose downstream package revisions need
to be updated. You can optionally pass in package revision names as arguments
in order to just list updates for those package revisions, or you can
pass in no arguments in order to list available updates for all package
revisions.

```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# update deployment-e982b2196b35a4f5e81e92f49a430fe463aa9f1a package to v3 of its upstream
$ kpt alpha rpkg update deployment-e982b2196b35a4f5e81e92f49a430fe463aa9f1a --revision=v3
```

```shell
# see available upstream updates for all your downstream packages
$ kpt alpha rpkg update --discover=upstream
```

```shell
# see available updates for any downstream packages that were created from the upstream blueprints-e982b2196b35a4f5e81e92f49a430fe463aa9f1a package
$ kpt alpha rpkg update --discover=downstream blueprints-e982b2196b35a4f5e81e92f49a430fe463aa9f1a
```

<!--mdtogo-->