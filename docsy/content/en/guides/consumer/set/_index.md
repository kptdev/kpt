---
title: "Set field values"
linkTitle: "Set"
weight: 3
type: docs
description: >
    Customize a local package by setting field values.
---

*Dynamic needs for packages are built into tools which read and write
configuration data.*

## Topics

[kpt cfg set], [setters], [Kptfile]

Kpt packages can be modified using existing tools and workflows such as
manually modifying the configuration in an editor, however these workflows
can be labor intensive and error prone.

To address the UX limitations of hand editing YAML, kpt provides built-in
commands which **expose setting values in a user friendly way from
the commandline**.

{{% pageinfo color="primary" %}}
Rather than exposing values as input parameters to a template,
commands **modify the package data in place**.

These commands are **defined per-package through OpenAPI definitions**
which are part of the package metadata -- i.e. the [Kptfile].

While OpenAPI is often used to define schema for static types
(e.g. this is what **a Deployment** looks like), kpt uses OpenAPI to define
**schema for individual instances of a type** as well
(e.g. this is what **the nginx Deployment** looks like).
{{% /pageinfo %}}

To see more on how to create a setter: [create setter guide]

## Setters explained

Following is a short explanation of the command that will be demonstrated
in this guide.

### Data model

- Fields reference setters through OpenAPI definitions specified as
  line comments -- e.g. `# { "$ref": "#/definitions/..." }`
- OpenAPI definitions are provided through the Kptfile

### Command control flow

1. Read the package Kptfile and resources.
2. Change the setter OpenAPI value in the Kptfile
3. Locate all fields which reference the setter and change their values.
4. Write both the modified Kptfile and resources back to the package.

{{< svg src="images/set-command" >}}

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [List the setters](#list-the-setters)
3. [Set a field](#set-a-field)

## Fetch a remote package

### Command

```sh
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.3.0 helloworld
```

### Output

```sh
fetching package /package-examples/helloworld-set from https://github.com/GoogleContainerTools/kpt to helloworld
```

## List the setters

The `helloworld-set` package contains [setters] which can be used to
**set configuration values from the commandline.**

##### Command

```sh
kpt cfg list-setters helloworld/ 
```

Print the list of setters included in the package.

##### Output

```sh
    NAME      VALUE       SET BY             DESCRIPTION        COUNT  
  http-port   80      package-default   helloworld port         3      
  image-tag   0.1.0   package-default   hello-world image tag   1      
  replicas    5       package-default   helloworld replicas     1     
```

The package contains 3 setters which may be used to modify the configuration
using `kpt set`.

## Set a field

Setters **modify the resource configuration in place by reading the resources,
changing values, and writing them back.**

##### Package contents

```yaml
# helloworld/deploy.yaml
kind: Deployment
metadata:
 name: helloworld-gke
...
spec:
 replicas: 5 # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
```

##### Command

```sh
kpt cfg set replicas 3
```

Change the replicas value in the configuration from 5 to 3.

##### Output

```sh
set 1 fields
```

##### Updated package contents

```yaml
kind: Deployment
metadata:
 name: helloworld-gke
...
spec:
 replicas: 3 # {"$ref":"#/definitions/io.k8s.cli.setters.replicas"}
...
```

[Kptfile]: ../../../api-reference/kptfile
[kpt cfg set]: ../../../reference/cfg/set
[setters]: ../../../reference/cfg/create-setter
[create setter guide]: ../../producer/setters
