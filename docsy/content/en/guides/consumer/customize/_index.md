---
title: "Customize a local package"
linkTitle: "Customize"
weight: 3
type: docs
description: >
    Customize a local package by modifying its contents.
---

*Data enables the creation of tools to read and write the YAML on behalf of
users.  Dynamic logic can still be developed, but loosely coupled to the data
that is applied to the cluster.*

## Topics

[kpt cfg set], [setters], [substitutions], [Kptfile]

Kpt packages can be modified using existing tools and workflows such as
manually modifying the configuration in an editor, however these workflows
can be labor intensive and error prone.

To address the UX limitations of hand editing YAML, kpt provides built-in
commands which can expose high-level values to set fields within the package.

{{% pageinfo color="primary" %}}
Rather than exposing high-level values as input parameters to a template,
these commands the ability to **modify the package data in place by setting
and substituting values**.

The configuration for these commands is **defined per-package
through OpenAPI definitions** included in the package metadata
(i.e. the [Kptfile].

While traditionally OpenAPI is used to define schema for static types
(e.g. this is what a Deployment looks like), kpt allows OpenAPI to also define
schema for individual instances of a type (e.g. this is what the nginx
Deployment looks like).
{{% /pageinfo %}}

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [List the setters](#list-the-setters)
3. [Set a field](#set-a-field)
4. [Substitute a value](#substitute-a-value)
5. [Edit configuration](#edit-configuration)

## Fetch a remote package

### Command

```sh
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.3.0 helloworld
```

Grab a remote package.  This guide will use one of the kpt package examples,
which contain additional metadata kpt can use to work with the package contents.

### Output

```sh
fetching package /package-examples/helloworld-set from https://github.com/GoogleContainerTools/kpt to helloworld
```

## List the setters

The `helloworld-set` package contains [setters] and [substitutions]
for modifying configuration from the commandline.

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

Setters modify the resource configuration in place by reading the resources,
changing values, and writing them back.

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

The package contents now contains the updated replicas value of 3.

## Substitute a value

Substitutions are performed through calling setters which are referenced by
substitutions.

{{% pageinfo color="primary" %}}
There is no command to list substitutions because they are not invoked directly,
but are instead performed when a setter referenced by the substitution is
invoked.

Substitutions can be found by looking in the [Kptfile] under
`openAPI.definitions`
{{% /pageinfo %}}

##### Package contents

```yaml
# helloworld/deploy.yaml
kind: Deployment
metadata:
 name: helloworld-gke
...
    spec:
      containers:
      - name: helloworld-gke
        image: gcr.io/kpt-dev/helloworld-gke:v0.1.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
...
```

##### Command

```sh
 kpt cfg set helloworld/ image-tag v0.2.0
```

Change the tag portion of the image field by setting the `image-tag` value to
`v0.2.0`.  This value will be substituted into a substitution pattern defined
in the `Kptfile` and written to the `image` field.

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
      containers:
      - name: helloworld-gke
        image: gcr.io/kpt-dev/helloworld-gke:v0.2.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
...
```

### Customizing setters

Because setters are defined using data as part of the package as OpenAPI data,
they don’t need to be compiled into the tool and can be customized
for each local instance of a package.

See [setters] and [substitutions] for how to add or update them in the
package [Kptfile].

[Kptfile]: ../../../api-reference/kptfile
[kpt cfg set]: ../../../reference/cfg/set
[setters]: ../../../reference/cfg/create-setter
[substitutions]: ../../../reference/cfg/create-subst
