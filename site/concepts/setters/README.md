---
title: "Setters"
linkTitle: "Setters"
weight: 4
type: docs
description: >
  Setters and related concepts
---

## Setters

Setters provide a solution for template-free setting or substitution of field
values through package metadata (OpenAPI). They are a safer alternative to other
substitution techniques which do not have the context of the structured data
– e.g. using `sed` to replace values.

The OpenAPI definitions for setters are defined in a Kptfile and referenced by a
fields through comments on the fields.

Setters may be invoked to modify the configuration using `kpt cfg set`
to set and/or substitute values.

### Auto setters

The values of few setters are auto-filled when the package is fetched(/updated).
Package consumers need not invoke `kpt cfg set` on them explicitly.

#### gcloud configs

This category of auto-setters derive the values from the output of `gcloud config list`
command. Following are the names of supported `gcloud` auto-setters:

```
gcloud.core.project
gcloud.project.projectNumber
gcloud.compute.region
gcloud.compute.zone
```

#### Inherit Setter Values from Parent Package

##### Notice: This is an experimental feature and is subjected to changes soon

When a remote kpt package is fetched(/updated) into local directory, kpt searches
for its closest parent directory(if any) with a `Kptfile` in the file system and
auto-fills matching setters in the fetched(/updated) package with the setter values
from the parent package. Setters are matched based on setter names.

e.g. Consider an example of `wordpress` package on local

```sh
wordpress
├── Kptfile
└── deployment.yaml
```

```sh
$ kpt cfg list-setters wordpress/
wordpress/
   NAME             VALUE     SET BY   DESCRIPTION   COUNT   REQUIRED   IS SET
 namespace         my-space                            1       No         Yes
```

If `mysql` package is fetched into the local directory tree of `wordpress`
package, and if both packages contain setter `namespace`, then the `namespace`
setter of `mysql` is automatically set to the value from `wordpress` package i.e.
`mysql` inherits setter value from `wordpress` package.

```sh
$ kpt pkg get git@github.com:example.git/mysql ./wordpress/
fetching package / from git@github.com:example.git/mysql to wordpress/mysql
automatically set 1 field(s) for setter "namespace" to value "my-namespace" in
package "wordpress/mysql" derived from parent "wordpress/Kptfile"
```

```sh
$ tree wordpress/
wordpress
├── Kptfile
├── mysql # subpackage
│   ├── Kptfile
│   └── deployment.yaml
└── deployment.yaml
```
