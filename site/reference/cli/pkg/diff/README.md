---
title: "`diff`"
linkTitle: "diff"
type: docs
description: >
  Show differences between a local package and upstream.
---

<!--mdtogo:Short
   Show differences between a local package and upstream.
-->

`diff` displays differences between upstream and local packages.

It can display differences between:

- The local package and the upstream version which the local package was based
  on.
- The local package and the latest version of the upstream package.

`diff` fetches the versions of a package that are needed, but it delegates
displaying the differences to a command line diffing tool. By default, the
'diff' command line tool is used, but this can be changed with either the
`diff-tool` flag or the `KPT_EXTERNAL_DIFF` env variable.

### Synopsis

<!--mdtogo:Long-->

```
kpt pkg diff [PKG_PATH@VERSION] [flags]
```

#### Args

```
PKG_PATH:
  Local package path to compare. diff will fail if the directory doesn't exist, or does not
  contain a Kptfile. Defaults to the current working directory.

VERSION:
  A git tag, branch, or commit. Specified after the local_package with @, for
  example my-package@master.
  Defaults to the local package version that was last fetched.
```

#### Flags

```
--diff-type:
  The type of changes to view (local by default). Following types are
  supported:

  local: Shows changes in local package relative to upstream source package
         at original version.
  remote: Shows changes in upstream source package at target version
          relative to original version.
  combined: Shows changes in local package relative to upstream source
            package at target version.
  3way: Shows changes in local package and source package at target version
        relative to original version side by side.

--diff-tool:
  Command line diffing tool ('diff' by default) for showing the changes.
  Note that it overrides the KPT_EXTERNAL_DIFF environment variable.

  # Show changes using 'meld' commandline tool.
  kpt pkg diff @master --diff-tool meld

--diff-tool-opts:
  Commandline options to use with the command line diffing tool.
  Note that it overrides the KPT_EXTERNAL_DIFF_OPTS environment variable.

  # Show changes using the diff command with recursive options.
  kpt pkg diff @master --diff-tool meld --diff-tool-opts "-r"
```

#### Environment Variables

```
KPT_EXTERNAL_DIFF:
  Commandline diffing tool ('diff; by default) that will be used to show
  changes.

  # Use meld to show changes
  KPT_EXTERNAL_DIFF=meld kpt pkg diff

KPT_EXTERNAL_DIFF_OPTS:
  Commandline options to use for the diffing tool. For ex.
  # Using "-a" diff option
  KPT_EXTERNAL_DIFF_OPTS="-a" kpt pkg diff --diff-tool meld

KPT_CACHE_DIR:
  Controls where to cache remote packages when fetching them.
  Defaults to <HOME>/.kpt/repos/
  On macOS and Linux <HOME> is determined by the $HOME env variable, while on
  Windows it is given by the %USERPROFILE% env variable.
```

<!--mdtogo-->

### Examples

{{% hide %}}

<!-- @makeWorkplace @verifyExamples-->

```
# Set up workspace for the test.
TEST_HOME=$(mktemp -d)
cd $TEST_HOME
```

<!-- @fetchPackage @verifyExamples-->

```shell
export SRC_REPO=https://github.com/GoogleContainerTools/kpt.git
kpt pkg get $SRC_REPO/package-examples/helloworld-set hello-world
cd hello-world
```

{{% /hide %}}

<!--mdtogo:Examples-->
<!-- @pkgDiff @verifyExamples-->

```shell
# Show changes in current package relative to upstream source package.
$ kpt pkg diff
```

<!--mdtogo-->
