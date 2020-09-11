---
title: 'Exporting a Cloud Build Pipeline'
linkTitle: 'Cloud Build'
weight: 4
type: docs
description: >
  Export a Cloud Build config file that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [Cloud Build]. This tutorial takes about 5 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*Unfamiliar with Cloud Build? Here is [Cloud Build Quickstarts]*.

On your local machine, create an empty directory:

```shell script
mkdir function-export-example
cd function-export-example
```

{{% pageinfo color="warning" %}}
All commands must be run at the root of this directory.
{{% /pageinfo %}}

Use `kpt pkg get` to fetch source files of this tutorial:

```shell script
# Fetch source files
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/function-export-blueprint example-package
```

Then you will get an `example-package` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

## Exporting a pipeline

```shell script
kpt fn export example-package --workflow cloud-build --output cloudbuild.yaml
```

Running this command will generate a `cloudbuild.yaml` like this:

```yaml
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - exmaple-package
```

## Integrating with your existing pipeline

Now you can manually copy and paste the generated content into your existing build config file.

If you do not have one yet, you can simply put the file in the root of your project. It is fully functional.

If you want to see the diff after running kpt functions, append a `kpt pkg diff` step to make your `cloudbuild.yaml` look like this:

```yaml
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - example-package
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - pkg
      - diff
      - example-package
      - --diff-tool
      - git
      - --diff-tool-opts
      - "--no-pager diff"
```

## Viewing the result on Cloud Build

Run this command will trigger a build:

```
gcloud builds submit .
```

Then you can view the result on [Build History].

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Submit again, then view how the pipeline fails on Cloud Build.

[Cloud Build]: https://cloud.google.com/cloud-build
[Cloud Build Quickstarts]: https://cloud.google.com/cloud-build/docs/quickstarts
[Build History]: https://console.cloud.google.com/cloud-build/builds
