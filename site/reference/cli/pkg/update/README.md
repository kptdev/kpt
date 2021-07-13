---
title: "`update`"
linkTitle: "update"
type: docs
description: >
  Apply upstream package updates.
---

<!--mdtogo:Short
    Apply upstream package updates.
-->

`update` pulls in upstream changes and merges them into a local package. Changes
may be applied using one of several strategies.

Since this will update the local package, all changes must be committed to git
before running `update`.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg update [PKG_PATH][@VERSION] [flags]
```

#### Args

```
PKG_PATH:
  Local package path to update. Directory must exist and contain a Kptfile
  to be updated. Defaults to the current working directory.

VERSION:
  A git tag, branch, ref or commit. Specified after the local_package
  with @ -- pkg@version.
  Defaults the ref specified in the Upstream section of the package Kptfile.

  Version types:
    * branch: update the local contents to the tip of the remote branch
    * tag: update the local contents to the remote tag
    * commit: update the local contents to the remote commit
```

#### Flags

```
--strategy:
  Defines which strategy should be used to update the package. This will change
  the update strategy for the current kpt package for the current and future
  updates. If a strategy is not provided, the strategy specified in the package
  Kptfile will be used.

    * resource-merge: Perform a structural comparison of the original /
      updated resources, and merge the changes into the local package.
    * fast-forward: Fail without updating if the local package was modified
      since it was fetched.
    * force-delete-replace: Wipe all the local changes to the package and replace
      it with the remote version.
```

#### Env Vars

```
KPT_CACHE_DIR:
  Controls where to cache remote packages when fetching them.
  Defaults to <HOME>/.kpt/repos/
  On macOS and Linux <HOME> is determined by the $HOME env variable, while on
  Windows it is given by the %USERPROFILE% env variable.
```

<!--mdtogo-->

### Examples

<!--mdtogo:Examples-->

```shell
# Update package in the current directory.
# git add . && git commit -m 'some message'
$ kpt pkg update
```

```shell
# Update my-package-dir/ to match the v1.3 branch or tag.
# git add . && git commit -m 'some message'
$ kpt pkg update my-package-dir/@v1.3
```

```shell
# Update with the fast-forward strategy.
# git add . && git commit -m "some message"
$ kpt pkg update my-package-dir/@master --strategy fast-forward
```

<!--mdtogo-->

### Details

#### Resource-merge strategy

The resource-merge strategy performs a structural comparison of each resource using the
OpenAPI schema. So rather than performing a text-based merge, kpt leverages the
common structure of KRM resources.

##### Resource identity
In order to perform a per-resource merge, kpt needs to be able to match a resource in
the local package with the same resource in the upstream version of the package. It does
this matching based on the identity of a resource, which is the combination of group,
kind, name and namespace. So in our wordpress example, the identity of the`Deployment`
resource is:
```
group: apps
kind: Deployment
name: wordpress
namespace: ""
```
Changing the name and/or namespace of a resource is a pretty common way to customize
a package. In order to make sure this doesn't create problems during merge, kpt will
automatically adding the `# kpt-merge: <namespace>/<name>` comment on the `metadata`
field of every resource when getting or updating a package. An example is the `Deployment`
resource from the wordpress package:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata: # kpt-merge: /wordpress
  name: wordpress
  labels:
    app: wordpress
...
```

##### Merge rules
kpt performs a 3-way merge for every resource. This means it will use the resource
in the local package, the updated resource from upstream, as well as the resource
at the version where the local and upstream package diverged (i.e.
common ancestor). When discussing the merge rules in detail, we will be referring to
the three different sources as local, upstream and origin.

In the discussion, we will be referring to non-associative and associative lists. A
non-associative list either has elements that are scalars or another list, or it has elements
that are mappings but without an associative key. An example of this in the kubernetes
API is the `command` property on containers:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod
spec:
  containers:
    - name: hello
      image: busybox
      command: ['sh', '-c', 'echo "Hello, World!"]
```

An associative list has elements that are mappings and
one or more of the fields in the mappings are designated as associative keys. An associative key
(also sometimes referred to as a merge key) is used to identify the "same" elements in two
different lists for the purpose of merging them. An example from the kubernetes API
is the list of containers in a pod which uses the `name` property as the merge key:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod
spec:
  containers:
    - name: web
      image: nginx
    - name: sidecar
      image: log-collector
```

kpt will primarily look for information about
any associative keys from the OpenAPI schema, but some fields are also automatically recognized as
associative keys:
* `mountPath`
* `devicePath`
* `ip`
* `type`
* `topologyKey`
* `name`
* `containerPort`

The 3-way merge algorithm operates both on the level of each resource and on
each individual field with a resource. 

On the resource level, the rules are:

* A resource present in origin and deleted from upstream will be deleted from local.
* A resource missing from origin and added in upstream will be added to local.
* A resource only in local will be kept without changes.
* A resource in both upstream and local will be merged into local.

On the field level, the rules differ based on the type of field.

For scalars and non-associative lists:
* If the field is present in either upstream or local and the value is `null`, remove the field from local.
* If the field is unchanged between upstream and local, leave the local value unchanged.
* If the field has been changed in both upstream and local, update local with the value from upstream.

For mappings:
* If the field is present in either upstream or local and the value is `null`, remove the field from local.
* If the field is present only in local, leave the local value unchanged.
* If the field is not present in local, add the delta between origin and upstream as the value in local.
* If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

For associative lists:
* If the field is present in either upstream or local and the value is `null`, remove the field from local.
* If the field is present only in local, leave the local value unchanged.
* If the field is not present in local, add the delta between origin and upstream as the value in local.
* If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

#### Fast-forward strategy

The fast-forward strategy updates a local package with the changes from upstream, but will
fail if the local package has been modified since it was fetched.

#### Force-delete-replace strategy

The force-delete-replace strategy updates a local package with changes from upstream, but will
wipe out any modifications to the local package.