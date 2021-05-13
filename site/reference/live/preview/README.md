---
title: "`preview`"
linkTitle: "preview"
type: docs
description: >
   Preview the changes apply would make to the cluster
---
<!--mdtogo:Short
    Preview the changes apply would make to the cluster
-->

`preview` shows the operations that will be done when running `apply` or 
`destroy` against a cluster.

### Synopsis
<!--mdtogo:Long-->
```
kpt live preview [PKG_PATH | -] [flags]
```

#### Args

```
PKG_PATH | -:
  Path to the local package for which a preview of the operations of apply
  or destroy should be displayed. It must contain a Kptfile with inventory
  information. Defaults to the current working directory.
  Using '-' as the package path will cause kpt to read resources from stdin.
```

#### Flags

```
--destroy:
  If true, preview deletion of all resources.

--field-manager:
  Identifier for the **owner** of the fields being applied. Only usable
  when --server-side flag is specified. Default value is kubectl.

--force-conflicts:
  Force overwrite of field conflicts during apply due to different field
  managers. Only usable when --server-side flag is specified.
  Default value is false (error and failure when field managers conflict).

--install-resource-group:
  Install the ResourceGroup CRD into the cluster if it isn't already
  available. Default is false.

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
  Determines the output format for the status information. Must be one of the following:
  
    * events: The output will be a list of the status events as they become available.
    * json: The output will be a list of the status events as they become available,
      each formatted as a json object.
    * table: The output will be presented as a table that will be updated inline
      as the status of resources become available.

  The default value is ‘events’.

--server-side:
  Perform the apply operation server-side rather than client-side.
  Default value is false (client-side).
```
<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->
```shell
# preview apply for the package in the current directory. 
kpt live preview
```

```shell
# preview destroy for a package in the my-dir directory.
kpt live preview --destroy my-dir
```
<!--mdtogo-->
