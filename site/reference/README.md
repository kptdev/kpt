---
title: "Command Reference"
linkTitle: "Command Reference"
type: docs
weight: 40
menu:
  main:
    weight: 3
description: >
  Overview of kpt commands
---

<!--mdtogo:Short
    Overview of kpt commands
-->

<!--mdtogo:Long-->

All kpt commands follow this general synopsis:

```
kpt <group> <command> [flags] <positional args> [PKG_PATH]
```

kpt functionality is divided into three command groups:

| Group   | Description                                                             |
| --------| ------------------------------------------------------------------------|
| pkg     | get, update, and describe packages with resources.                      |
| fn      | generate, transform, validate packages using containerized functions.   |
| live    | deploy local configuration packages to a cluster.                       |



<!--mdtogo-->
