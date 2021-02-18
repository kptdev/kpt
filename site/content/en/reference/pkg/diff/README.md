---
title: "Diff"
linkTitle: "diff"
type: docs
description: >
   Diff a local package against upstream
---
<!--mdtogo:Short
    Diff a local package against upstream
-->

Diff displays differences between upstream and local packages.

It can display differences between:

- local package and upstream source version
- local package and upstream new version

The diff tool can be specified.  By default, the local 'diff' command is used to
display differences.

### Examples
<!--mdtogo:Examples-->
```sh
# Show changes in current package relative to upstream source package
kpt pkg diff
```

```sh
# Show changes in current package relative to upstream source package
# using meld tool with auto compare option.
kpt pkg diff --diff-tool meld --diff-tool-opts "-a"
```

```sh
# Show changes in upstream source package between current version and
# target version
kpt pkg diff @v4.0.0 --diff-type remote
```

```sh
# Show changes in current package relative to target version
kpt pkg diff @v4.0.0 --diff-type combined
```

```sh
# Show 3way changes between the local package, upstream package at original
# version and upstream package at target version using meld
kpt pkg diff @v4.0.0 --diff-type 3way --diff-tool meld --diff-tool-opts "-a"
```
<!--mdtogo-->

### Synopsis
<!--mdtogo:Long-->
```
kpt pkg diff [DIR@VERSION]
```

#### Args

```
DIR:
  Local package to compare. Command will fail if the directory doesn't exist, or does not
  contain a Kptfile.  Defaults to the current working directory.

VERSION:
  A git tag, branch, ref or commit. Specified after the local_package with @ -- pkg_dir@version.
  Defaults to the local package version that was last fetched.
```

#### Flags

```
--diff-type:
  The type of changes to view (local by default). Following types are
  supported:

  local: shows changes in local package relative to upstream source package
         at original version
  remote: shows changes in upstream source package at target version
          relative to original version
  combined: shows changes in local package relative to upstream source
            package at target version
  3way: shows changes in local package and source package at target version
        relative to original version side by side

--diff-tool:
  Commandline tool (diff by default) for showing the changes.
  Note that it overrides the KPT_EXTERNAL_DIFF environment variable.
  
  # Show changes using 'meld' commandline tool
  kpt pkg diff @master --diff-tool meld

--diff-opts:
  Commandline options to use with the diffing tool.
  Note that it overrides the KPT_EXTERNAL_DIFF_OPTS environment variable.
  # Show changes using "diff" with recurive options
  kpt pkg diff @master --diff-tool meld --diff-opts "-r"
```

#### Environment Variables

```
KPT_EXTERNAL_DIFF:
   Commandline diffing tool (diff by default) that will be used to show
   changes.
   # Use meld to show changes
   KPT_EXTERNAL_DIFF=meld kpt pkg diff

KPT_EXTERNAL_DIFF_OPTS:
   Commandline options to use for the diffing tool. For ex.
   # Using "-a" diff option
   KPT_EXTERNAL_DIFF_OPTS="-a" kpt pkg diff --diff-tool meld
```
<!--mdtogo-->
