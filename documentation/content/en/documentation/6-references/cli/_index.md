---
title: CLI Reference
linkTitle: CLI Reference
description: Overview of kpt commands
aliases:
  - /reference/cli/
---

<!--mdtogo:Short
    Overview of kpt commands
-->

<!--mdtogo:Long-->

All kpt commands follow this general synopsis:

```shell
kpt <group> <command> <positional args> [PKG_PATH] [flags]
```

kpt functionality is divided into following command groups:

| Group   | Description                                                           |
| ------- | --------------------------------------------------------------------- |
| [pkg]   | get, update, and describe packages with resources.                    |
| [fn]    | generate, transform, validate packages using containerized functions. |
| [live]  | deploy local configuration packages to a cluster.                     |
| [alpha] | commands currently in alpha and might change without notice.          |

<!--mdtogo-->

[pkg]: /documentation/6-references/cli/pkg/
[fn]: /documentation/6-references/cli/fn/
[live]: /documentation/6-references/cli/live/
[alpha]: /documentation/6-references/cli/alpha/