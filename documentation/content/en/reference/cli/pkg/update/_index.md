---
title: "`update`"
linkTitle: "update"

description: |
  Apply upstream package updates.
---

<!--mdtogo:Short
    Apply upstream package updates.
-->

`update` pulls in upstream changes and merges them into a local package. Changes
may be applied using one of several strategies.

Since this will update the local package, all changes must be committed to git
before running `update`.

> **Want to understand 3-way merge in detail?** See the [3-Way Merge Guide]({{% relref "/guides/3-way-merge" %}}) for comprehensive documentation on how kpt merges packages, merge strategies, conflict resolution, and best practices.

### Synopsis

<!--mdtogo:Long-->

```shell
kpt pkg update [PKG_PATH][@VERSION] [flags]
```

#### Args

```shell
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

```shell
--strategy:
  Defines which strategy should be used to update the package. This will change
  the update strategy for the current kpt package for the current and future
  updates. If a strategy is not provided, the strategy specified in the package
  Kptfile will be used.

    * resource-merge: Perform a structural comparison of the original /
      updated resources, and merge the changes into the local package.
    * copy-merge: Replace local files with the upstream version at the file
      level, keeping files that were added purely locally.
    * fast-forward: Fail without updating if the local package was modified
      since it was fetched.
    * force-delete-replace: Wipe all the local changes to the package and replace
      it with the remote version.

--preserve-nulls:
  (Experimental) When set, null fields in the local package or upstream (`field:`, `null`, or
  `~`) are kept during resource-merge instead of being deleted. This changes
  the default for the current and future updates and is persisted as
  `upstream.preserveNulls` in the Kptfile.
  Only applies to the resource-merge strategy. Defaults to false.
```

#### Env Vars

```shell
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
When `--preserve-nulls` (or `upstream.preserveNulls` in the
Kptfile) is set, a null in local or upstream is kept instead of being
removed. This includes `null`, `~`, and an empty value (`field:`), and
applies to scalars, mappings, and lists.

For scalars and non-associative lists:
* Unless `--preserve-nulls` is set, a field present in either upstream or local whose value is `null`, `~`, or empty is removed from local.
* If the field is unchanged between upstream and local, leave the local value unchanged.
* If the field has been changed in both upstream and local, update local with the value from upstream.

When a field is changed in both upstream and local (a conflict), resource-merge
does not stop or emit conflict markers. It auto-resolves by always choosing the
new upstream value, and the update succeeds. If you need to keep the local value
for such a field, re-apply it after the update.

For mappings:
* Unless `--preserve-nulls` is set, a field present in either upstream or local whose value is `null`, `~`, or empty is removed from local.
* If the field is present only in local, leave the local value unchanged.
* If the field is not present in local, add the delta between origin and upstream as the value in local.
* If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

For associative lists:
* Unless `--preserve-nulls` is set, a field present in either upstream or local whose value is `null`, `~`, or empty is removed from local.
* If the field is present only in local, leave the local value unchanged.
* If the field is not present in local, add the delta between origin and upstream as the value in local.
* If the field is present in both upstream and local, recursively merge the values between local, upstream and origin.

#### Copy-merge strategy

The copy-merge strategy is a file-level replacement rather than a structural,
field-level merge. For most files that exist in both the local and upstream
packages, the upstream version replaces the local one, so in-file local edits to
upstream-owned files are lost.

There are two cases where a local file is NOT overwritten:

* The **root Kptfile** is an exception: copy-merge 3-way merges it (via the same
  Kptfile merge used by resource-merge) instead of overwriting it, so local
  customizations in the root Kptfile are preserved.
* Files that were **added purely locally** and never existed in the upstream
  package the local package was cloned from are kept.

Deletions when upstream removes a file follow the same ownership rule:

* A file that originated upstream and was later modified locally is still treated
  as upstream-owned. If upstream deletes it, it is deleted from local and the
  local modifications are lost.
* A file that was added only locally is kept, even if upstream deletes the
  directory now containing it (the containing directory is preserved too; only
  the upstream-owned files inside it are removed).

Use copy-merge when you trust the upstream content over local edits, or when kpt
cannot structurally parse the files (for example, non-KRM files). If you need to
preserve local modifications to upstream-owned files, use resource-merge instead.

#### Fast-forward strategy

The fast-forward strategy updates a local package with the changes from upstream, but will
fail if the local package has been modified since it was fetched.

Render status (`status.renderStatus` and the `Rendered` condition in `status.conditions`) written by `kpt fn render`
is not considered a local modification. It is automatically cleared from the local Kptfile after a successful
fast-forward update.

#### Force-delete-replace strategy

The force-delete-replace strategy updates a local package with changes from upstream, but will
wipe out any modifications to the local package.