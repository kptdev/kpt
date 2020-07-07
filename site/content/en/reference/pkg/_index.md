---
title: "Pkg"
linkTitle: "pkg"
weight: 1
type: docs
description: >
   Fetch, update, and sync configuration files using git
---
<!--mdtogo:Short
    Fetch, update, and sync configuration files using git
-->

{{< asciinema key="pkg" rows="10" preload="1" >}}

<!--mdtogo:Long-->
|              Reads From | Writes To                |
|-------------------------|--------------------------|
| git repository          | local directory          |

The `pkg` command group contains subcommands which read remote upstream
git repositories, and write local directories.  They are focused on
providing porcelain on top of workflows which would otherwise require
wrapping git to pull clone subdirectories and perform updates by merging
resources rather than files.
<!--mdtogo-->

    kpt pkg [SUBCOMMAND]

### Examples
<!--mdtogo:Examples-->
```sh
# create your workspace
$ mkdir hello-world-workspace
$ cd hello-world-workspace
$ git init

# get the package
$ export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
$ kpt pkg get $SRC_REPO/package-examples/helloworld-set@v0.3.0 helloworld

# add helloworld to your workspace
$ git add .
$ git commit -am "Add hello world to my workspace."

# pull in upstream updates by merging Resources
$ kpt pkg update helloworld@v0.5.0 --strategy=resource-merge
```
<!--mdtogo-->

### Synopsis

#### Package Format

1. **Any git repository containing resource configuration files may be used as a package**, no
   additional structure or formatting is necessary.
2. **Any package may be applied with `kubectl apply -R -f`**.
3. Packages **may be customized in place either manually (e.g. with `vi`) or programmatically**.
4. Packages **must** be worked on within a local git repo.

#### Model

1. **Packages are simply subdirectories of resource configuration files in git**
    * They may also contain supplemental non-resource artifacts, such as markdown files, templates, etc.
    * The ability to fetch a subdirectory of a git repo is a key difference compared to
      [git subtree](https://github.com/git/git/blob/master/contrib/subtree/git-subtree.txt).

2. **Any existing git subdirectory containing resource configuration files may be used as a package**
    * Nothing besides a git directory containing resource configuration is required.
    * e.g. any [example in the examples repo](https://github.com/kubernetes/examples/tree/master/staging/cockroachdb) may
      be used as a package:

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb \
            my-cockroachdb
          kubectl apply -R -f my-cockroachdb

3. **Packages should use git references for versioning**.
    * We recommend package authors use semantic versioning when publishing packages for others to consume.

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb@VERSION \
            my-cockroachdb
          kubectl apply -R -f my-cockroachdb

4. **Packages may be modified or customized in place**.
    * It is possible to directly modify the fetched package and merge upstream updates.

5. **The same package may be fetched multiple times** to separate locations.
    * Each instance may be modified and updated independently of the others.

          # fetch an instance of a java package
          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb db1
          # make changes...

          # fetch a second instance of a java package
          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb db2
          # make changes...

6. **Packages may pull upstream updates after they have been fetched and modified**.
    * Specify the target version to update to, and an (optional) update strategy for how to apply the
      upstream changes.

          kpt pkg get https://github.com/kubernetes/examples/staging/cockroachdb \
            my-cockroachdb
          # make changes...
          kpt pkg update my-cockroachdb@NEW_VERSION --strategy=resource-merge

7. **Packages must be customized and updated within a git repo**.
    * kpt facilitates configuration reuse. Key to that reuse is git. Git is used
      both as a unit to define the boundary of a git repo, but also as the
      boundary of a local workspace. In this way, any workspace is in itself a
      package. Local customizations of packages (or use of specific versions)
      can be re-published as the new canonical package for other users. kpt
      requires any customization to be committed to git before package updates
      can be reconciled.
