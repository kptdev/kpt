---
title: "`install-resource-group`"
linkTitle: "install-resource-group"
type: docs
description: >
Install the ResourceGroup CRD in the cluster.
---

<!--mdtogo:Short
    Install the ResourceGroup CRD in the cluster.
-->

`install-resource-group` installs the ResourceGroup CRD in the cluster. `kpt`
uses ResourceGroup resources for storing the inventory list (which enables
pruning) , so it must be installed in a cluster prior to applying packages.

### Synopsis

<!--mdtogo:Long-->

```
kpt live install-resource-group
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# install ResourceGroup CRD into the current cluster.
$ kpt live install-resource-group
```

<!--mdtogo-->
