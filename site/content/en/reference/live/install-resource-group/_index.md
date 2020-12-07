---
title: "Install-resource-group"
linkTitle: "install-resource-group"
type: docs
description: >
   Apply ResourceGroup custom resource definition to the cluster
---
<!--mdtogo:Short
    Apply ResourceGroup custom resource definition to the cluster
-->

NOTE: This command is not available unless the RESOURCE_GROUP_INVENTORY
environment variable is set.

The install-resource-group command applies the ResourceGroup
custom resource definition (CRD) to the cluster. This CRD allows
ResourceGroup custom resources to be created, acting as inventory
objects. The ResourceGroup custom resource is the next generation
inventory object after the ConfigMap.

### Examples
<!--mdtogo:Examples-->
```sh
# install the ResourceGroup CRD
export RESOURCE_GROUP_INVENTORY=1
kpt live install-resource-group
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live install-resource-group
```

#### Args

None

#### Flags

None

<!--mdtogo-->
