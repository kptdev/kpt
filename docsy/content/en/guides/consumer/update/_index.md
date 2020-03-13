---
title: "Update a local package"
linkTitle: "Update"
weight: 2
type: docs
description: >
    Update a customized local package with upstream (remote) package changes.
---

## Topics

[kpt pkg update]

Because kpt package contents are resource configuration (data) rather
than templates or DSLs (code), it is possible to merge different versions
of the package together using the structure of the resources to compute
differences.

This allows package consumers to customize their copy, and merge updates
from upstream.

{{% pageinfo color="primary" %}}
The technique of merging fields to perform updates is also how `kubectl apply`
updates remote cluster resources with local file changes, without overwriting
changes to the resources made by the cluster control-plane (e.g. an autoscaler
can set replicas without apply overwriting them). 

See [update strategies] for more choices on how to merge upstream changes.
{{% /pageinfo %}}


## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [Edit the contents](#edit-the-contents)
3. [Commit local changes](#commit-local-changes)
4. [Merge upstream changes](#merge-upstream-changes)
5. [View new package contents](#view-new-package-contents)

## Fetch a remote package

Packages can be fetched at specific versions defined by git tags, and have
updated to later versions to merge in upstream changes.

##### Command

```sh
export REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $REPO/package-examples/helloworld-set@v0.3.0 helloworld
```

Fetch the `helloworld-set` package at version `v0.3.0`.

##### Output

```
fetching package /package-examples/helloworld-set from https://github.com/GoogleContainerTools/kpt to helloworld
```

{{% pageinfo color="info" %}}
Each subdirectory within a git repo may be tagged with its own version
using the subdirectory path as a tag prefix, and kpt will automatically
resolve the subdirectory version.

`package-examples/helloworld-set@v0.3.0` is resolved to the tag
`package-examples/helloworld-set/v0.3.0` if it exists, otherwise it is
resolved to the tag `v0.3.0`.
{{% /pageinfo %}}

## Edit the contents

Edit the contents of the package by making changes to it.

##### Old local configuration

```yaml
# helloworld/deploy.yaml (upstream)
...
        image: gcr.io/kpt-dev/helloworld-gke:v0.1.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
...
        env:
        - name: PORT
          value: "80" # {"$ref":"#/definitions/io.k8s.cli.setters.http-port"}
...
```

The old package contents without local modifications.

```sh
vi helloworld/deploy.yaml
```

#####  New local configuration

```yaml
# helloworld/deploy.yaml (locally modified)
...
        image: gcr.io/kpt-dev/helloworld-gke:v0.1.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
...
        env:
        - name: PORT
          value: "80" # {"$ref":"#/definitions/io.k8s.cli.setters.http-port"}
        - name: NEW_ENV # This is a local package addition
          value: "local package edits"
...
```

The new package contents with local modifications.

## Commit local changes

{{% pageinfo color="warning" %}}
In order for updates to be easily undone, configuration must be
committed to git prior to performing a package update.

kpt will throw an error if trying to update a package and the git repo
has uncommitted changes.
{{% /pageinfo %}}

```sh
git init
git add .
git commit -m "add helloworld package at v0.3.0"
```

## Merge upstream changes

Package updates are performed by fetching the upstream package at the
specified version and applying the upstream changes to the local package.

##### Command

```sh
kpt pkg update helloworld-set@v0.5.0 --strategy=resource-merge
```

Update the local package to the upstream version v0.5.0 by doing a 3-way
merge between 1) the original upstream commit, 2) the local (customized)
package, 3) the new upstream reference.

##### Output

```sh
updating package helloworld to v0.5.0
```

##### Changes

```sh
--- a/helloworld/deploy.yaml
+++ b/helloworld/deploy.yaml
@@ -31,7 +31,7 @@ spec:
     spec:
       containers:
       - name: helloworld-gke
-        image: gcr.io/kpt-dev/helloworld-gke:0.1.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
+        image: gcr.io/kpt-dev/helloworld-gke:v0.3.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
         ports:
         - name: http
           containerPort: 80 # {"$ref":"#/definitions/io.k8s.cli.setters.http-port"}
diff --git a/helloworld/service.yaml b/helloworld/service.yaml
index 0853ee1..c938fde 100644
--- a/helloworld/service.yaml
+++ b/helloworld/service.yaml
@@ -22,7 +22,7 @@ metadata:
   labels:
```

The Deployment was updated with a new image tag.

```sh
--- a/helloworld/service.yaml
+++ b/helloworld/service.yaml
@@ -22,7 +22,7 @@ metadata:
   labels:
     app: hello
 spec:
-  type: LoadBalancer
+  type: NodePort
   selector:
     app: hello
   ports:
```

The Service was updated with a new `type`.

```sh
--- a/helloworld/Kptfile
+++ b/helloworld/Kptfile
@@ -5,10 +5,10 @@ metadata:
 upstream:
     type: git
     git:
-        commit: 3d721bafd701deb06aeb43c5ea5afda3134cfdd6
+        commit: 3f173ad974081896b47f6929b2c3cb595d71af94
         repo: https://github.com/GoogleContainerTools/kpt
         directory: /package-examples/helloworld-set
-        ref: v0.3.0
+        ref: v0.5.0
 openAPI:
     definitions:
         io.k8s.cli.setters.http-port:
```

The Kptfile was updated with the new upstream metadata.

## View new package contents

```yaml
# helloworld/deploy.yaml (updated from upstream)
...
        image: gcr.io/kpt-dev/helloworld-gke:v0.3.0 # {"$ref":"#/definitions/io.k8s.cli.substitutions.image-tag"}
...
        env:
        - name: PORT
          value: "80" # {"$ref":"#/definitions/io.k8s.cli.setters.http-port"}
        - name: NEW_ENV # This is a local package addition
          value: "local package edits"
...
```

The updated local package contains *both* the upstream changes (new image tag),
and local modifications (additional environment variable).

[kpt pkg update]: ../../../reference/pkg/update
[update strategies]: ../../../reference/pkg/update/#flags
