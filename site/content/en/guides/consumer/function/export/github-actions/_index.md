---
title: 'Exporting a GitHub Actions Workflow'
linkTitle: 'GitHub Actions'
type: docs
description: >
  Export a GitHub Actions workflow that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a workflow that runs the functions against the resources on [GitHub Actions](https://github.com/features/actions) and modify it manually to make it fully functional. This tutorial takes about 10 minutes.

## Before you begin

Before diving into the following tutorial, you may need to create a public repo on GitHub if you don't have one yet, e.g. `function-export-example`.

On your local machine, create an empty directory:

```shell script
mkdir function-export-example
cd function-export-example
```

All commands must be run at the root of this directory.

Use `kpt pkg get` to fetch source files of this tutorial:

```shell script
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/function-export-blueprint exmaple-package
# Init git
git init
git remote add origin https://github.com/<USER>/<REPO>.git
```

Then you will get an `exmaple-package` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions from [Kpt Functions Catalog](../../catalog) declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

## Exporting a workflow

```shell script
kpt fn export exmaple-package --workflow github-actions --output main.yaml
```

Running the command above will produce a `main.yaml` file that looks like this:

```yaml
name: kpt
on:
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - name: Run all kpt functions
            uses: docker://gcr.io/kpt-dev/kpt:latest
            with:
                args: fn run exmaple-package
```

## Integrating with your existing pipeline

Now you can manually copy and paste the content of the `main.yaml` file into your existing GitHub Actions workflow.
If you do not have one, you can copy the content of the exported `main.yaml` file into `.github/workflows/main.yaml` in your project root. To make it fully functional, you may add a `checkout` step before the `Run all kpt functions` step to pull source files from your repo:

```yaml
name: kpt
on:
    push:
        branches:
          - master
jobs:
    Kpt:
        runs-on: ubuntu-latest
        steps:
          - uses: actions/checkout@v2

          - name: Run all kpt functions
            uses: docker://gcr.io/kpt-dev/kpt:latest
            with:
                args: fn run exmaple-package
```

## Viewing the result on GitHub Actions

Once the changes are committed and pushed, you can see the latest job on GitHub Actions like this:

{{< png src="images/fn-export/github-actions-result" >}}
