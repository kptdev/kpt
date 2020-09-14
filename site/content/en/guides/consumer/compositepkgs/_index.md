---
title: "Get and apply a remote composite package"
linkTitle: "Composite Package"
weight: 7
type: docs
description: >
  Fetch a composite package from a remote git repository and apply its contents to
  a cluster
---

{{% pageinfo color="warning" %}}

#### Notice: Composite packages feature support is in alpha phase

{{% /pageinfo %}}

This guide walks you through an example to get, view, set and apply contents of a
kpt package which has subpackages in it. A kpt package is a directory of
resource configs with a valid `Kptfile` in it. A composite package is a `kpt` package
with 1 or more subpackages in its directory tree.

Principles:

1. Each kpt package is an independent building block and
   should contain resources(ex: setter definitions) of its own.
2. If a package is present in the directory tree of parent package,
   the configs of that package are out of scope for the actions performed
   on the parent package.
3. To run a command recursively on all the subpackages, users can leverage
   `--recurse-subpackages(-R)` flag. This is equivalent to running the same
   command on each package path in the directory tree.

## Steps

1. [Fetch a remote package](#fetch-a-remote-package)
2. [View the package contents](#view-the-package-contents)
3. [Set the setter parameters](#set-the-setter-parameters)
4. [Apply the composite package](#apply-the-composite-package)

## Fetch a remote package

Fetch an example package which has subpackages using [kpt pkg get].

### get command

```sh
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/hello-composite-pkg \
hello-composite-pkg
```

## View the package contents

The primary package artifacts are Kubernetes resource configuration
(e.g. YAML files).

### tree command

[tree] command is used to list the contents of the package in a tree structure.

```sh
kpt cfg tree hello-composite-pkg/
```

Output:

```sh
hello-composite-pkg
├── [Kptfile]  Kptfile hello-composite-pkg
├── [deploy.yaml]  Deployment default/hello-composite
└── Pkg: hello-subpkg
    ├── [Kptfile]  Kptfile hello-subpkg
    ├── [deploy.yaml]  Deployment default/hello-sub
    ├── hello-dir
    │   └── [configmap.yaml]  ConfigMap default/hello-cm
    └── Pkg: hello-nestedpkg
        ├── [Kptfile]  Kptfile hello-nestedpkg
        └── [deploy.yaml]  Deployment default/hello-nested
```

There are three kpt packages in the output:

1. hello-composite-pkg
2. hello-subpkg
3. hello-nestedpkg

`hello-dir` is a subdirectory of `hello-subpkg` and is not a kpt package, as it doesn't
contain a `Kptfile` in it.

Optionally, users may use other commands like [count], [grep], [cat] to
further view and understand the package contents.

### list-setters command

The fetched package contains parameters called setters which can be used to set configuration
values from the commandline.

```sh
kpt cfg list-setters hello-composite-pkg/
```

Prints the list of setters included recursively in all the subpackages

Output:

```sh
hello-composite-pkg/
         NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
  gcloud.core.project   YOUR_PROJECT_ID                          1       No
  image                 helloworld-gke                           1       No
  namespace             YOURSPACE                                1       Yes
  tag                   0.1.0                                    1       No

hello-composite-pkg/hello-subpkg/
         NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
  gcloud.core.project   YOUR_PROJECT_ID                          1       No
  image                 helloworld-gke                           1       No
  namespace             YOURSPACE                                2       Yes
  tag                   0.1.0                                    1       No

hello-composite-pkg/hello-subpkg/hello-nestedpkg/
         NAME                VALUE        SET BY   DESCRIPTION   COUNT   REQUIRED
  gcloud.core.project   YOUR_PROJECT_ID                          1       No
  image                 helloworld-gke                           1       No
  namespace             YOURSPACE                                1       Yes
  tag                   0.1.0                                    1       No
```

If you have `gcloud` set up on your local, you can observe that the value of the setter
`gcloud.core.project` is set automatically when the package is fetched.  
[Auto-setters] are automatically set deriving the values from the output of
`gcloud config list` command, when the package is fetched using [kpt pkg get].

## Provide the setter values

Set operation modifies the resource configuration in place by reading the resources,
changing parameter values, and writing them back.

In `list-setters` output, `namespace` setter is marked as required by the package
publisher, hence it is mandatory to set it to a new value. You may set other setter
parameters, either selectively in few of the packages or in all the them.

`--recurse-subpackages(-R)` flag is `default:false` for set command. If not invoked,
the set operation is performed only on the resource files of parent package and not
the subpackages.

```sh
kpt cfg set hello-composite-pkg/ namespace myspace -R
```

Output:

```sh
hello-composite-pkg/
set 1 field(s)

hello-composite-pkg/hello-subpkg/
set 2 field(s)

hello-composite-pkg/hello-subpkg/hello-nestedpkg/
set 1 field(s)
```

## Apply the composite package

Now that you have configured the package, apply it to the cluster

```sh
kubectl create namespace myspace
kubectl apply -f hello-composite-pkg/ -R
```

Output:

```sh
deployment.apps/hello-composite created
deployment.apps/hello-sub created
configmap/hello-cm created
deployment.apps/hello-nested created
```

[kpt pkg get]: ../../..//reference/pkg/get/
[tree]: https://googlecontainertools.github.io/kpt/reference/cfg/tree/
[count]: https://googlecontainertools.github.io/kpt/reference/cfg/count/
[grep]: https://googlecontainertools.github.io/kpt/reference/cfg/grep/
[cat]: https://googlecontainertools.github.io/kpt/reference/cfg/cat/
[auto-setters]: https://googlecontainertools.github.io/kpt/guides/producer/setters/#auto-setters
