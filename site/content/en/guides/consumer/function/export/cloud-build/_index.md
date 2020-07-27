---
title: 'Exporting a Cloud Build Pipeline'
linkTitle: 'Cloud Build'
type: docs
description: >
  Export a Cloud Build config file that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [Cloud Build](https://cloud.google.com/cloud-build). This tutorial takes about 5 minutes.

## Before you begin

Before diving into the following tutorial, you need to create a public repo on GitHub, e.g. `function-export-example`.

On your local machine, use `kpt pkg get` to fetch source files of this tutorial:

```shell script
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/function-export-blueprint function-export-example
cd function-export-example
# Init git
git init
git remote add origin https://github.com/<USER>/<REPO>.git
```

Then you will get a `function-export-example` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions from [Kpt Functions Catalog](../../catalog) declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

All commands must be run at the root of this directory.

## Exporting a pipeline

```shell script
kpt fn export \
    resources \
    --fn-path functions.yaml \
    --workflow cloud-build \
    --output cloudbuild.yaml
```

Running this command will generate a `cloudbuild.yaml` like this:

```yaml
steps:
  - name: gcr.io/kpt-dev/kpt:latest
    args:
      - fn
      - run
      - resources
      - --fn-path
      - functions.yaml
```

## Integrating with your existing pipeline

Now you can manually copy and paste the generated content into your existing build config file.

If you do not have one yet, you can simply put the file in the root of your project. It is fully functional.

## Viewing the result on Cloud Build

Run this command will trigger a build:

```
gcloud builds submit --config cloudbuild.yaml
```

Then you can view the result on [Cloud Build](https://console.cloud.google.com/cloud-build/builds).
