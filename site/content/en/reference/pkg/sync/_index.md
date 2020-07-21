---
title: "Sync"
linkTitle: "sync"
type: docs
draft: true
description: >
   Fetch and update packages declaratively
---
<!--mdtogo:Short
    Fetch and update packages declaratively
-->

{{< asciinema key="pkg-sync" rows="10" preload="1" >}}

Sync gets and updates packages using a manifest to manage a collection
of dependencies.

The manifest declares *all* direct dependencies of a package in a Kptfile.
When `sync` is run, it will ensure each dependency has been fetched at the
specified ref.

This is an alternative to managing package dependencies individually using
the `get` and `update` commands.

### Examples

#### Example sync commands
<!--mdtogo:Examples-->
```sh
# print the dependencies that would be modified
kpt pkg sync . --dry-run
```

```sh
# sync the dependencies
kpt pkg sync .
```
<!--mdtogo-->

#### Example Kptfile with dependencies

```yaml
# file: Kptfile
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
# list of dependencies to sync
dependencies:
- name: local/destination/dir
  git:
    # repo is the git respository
    repo: "https://github.com/pwittrock/examples"
    # directory is the git subdirectory
    directory: "staging/cockroachdb"
    # ref is the ref to fetch
    ref: "v1.0.0"
- name: local/destination/dir1
  git:
    repo: "https://github.com/pwittrock/examples"
    directory: "staging/javaee"
    ref: "v1.0.0"
  # set the strategy for applying package updates
  updateStrategy: "resource-merge"
- name: app2
  path: local/destination/dir2
  # declaratively delete this dependency
  ensureNotExists: true
```

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg sync LOCAL_PKG_DIR [flags]

LOCAL_PKG_DIR:
  Local package with dependencies to sync.  Directory must exist and
  contain a Kptfile.
```

#### Env Vars

```
KPT_CACHE_DIR:
  Controls where to cache remote packages during updates.
  Defaults to ~/.kpt/repos/
```
<!--mdtogo-->

#### Dependencies

For each dependency in the Kptfile, `sync` will ensure that it exists
locally with the matching repo and ref.

Dependencies are specified in the Kptfile `dependencies` field and can be
added or updated with `kpt pkg sync set`.  e.g.

```sh
kpt pkg sync set https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set \
    hello-world
```

The [sync-set] command must be run from within the local package directory and the
last argument specifies the local destination directory for the dependency.

Or edit the Kptfile directly:

```yaml
apiVersion: kpt.dev/v1alpha1
kind: Kptfile
dependencies:
- name: hello-world
  git:
    repo: "https://github.com/GoogleContainerTools/kpt.git"
    directory: "/package-examples/helloworld-set"
    ref: "master"
```

Dependencies have following schema:

```yaml
name: <local path (relative to the Kptfile) to fetch the dependency to>
git:
  repo: <git repository>
  directory: <sub-directory under the git repository>
  ref: <git reference -- e.g. tag, branch, commit, etc>
updateStrategy: <strategy to use when updating the dependency -- see kpt help update for more details>
ensureNotExists: <remove the dependency, mutually exclusive with git>
```

Dependencies maybe be updated by updating their `git.ref` field and running `kpt pkg sync`
against the directory.

[sync-set]: set
