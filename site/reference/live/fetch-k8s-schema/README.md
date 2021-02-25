---
title: "Fetch-k8s-schema"
linkTitle: "fetch-k8s-schema"
type: docs
description: >
   Fetch the OpenAPI schema from the cluster
---
<!--mdtogo:Short
    Fetch the OpenAPI schema from the cluster
-->

The fetch-k8s-schema command downloads the OpenAPI schema from the cluster
given by the context. It prints the result to stdout.

### Examples
<!--mdtogo:Examples-->
```sh
# print the schema for the cluster given by the current context
kpt live fetch-k8s-schema

# print the schema after formatting using a named context
kpt live fetch-k8s-schema --context=myContext --pretty-print
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt live fetch-k8s-schema [flags]
```

#### Flags

```
--pretty-print
  Format the output before printing
```
<!--mdtogo-->
