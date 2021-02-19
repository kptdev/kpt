---
title: 'Exporting a GitHub Actions Workflow'
linkTitle: 'GitHub Actions'
weight: 1
type: docs
description: >
  Export a GitHub Actions workflow that runs kpt functions
---

In this tutorial, you will pull an example package that declares Kubernetes resources and two kpt functions. Then you will export a workflow that runs the functions against the resources on [GitHub Actions] and modify it manually to make it fully functional. This tutorial takes about 10 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*New to GitHub Actions? Here is [How to Configure a Workflow]*.

Before diving into the following tutorial, you need to create a public repo on GitHub if you don't have one yet, e.g. `function-export-example`.

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
# Init git
git init
git remote add origin https://github.com/<USER>/<REPO>.git
# Fetch source files
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/function-export example-package
```

Then you will get an `example-package` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

## Exporting a workflow

```shell script
kpt fn export example-package --workflow github-actions --output main.yaml
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
                args: fn run example-package
```

## Integrating with your existing pipeline

Now you can manually copy and paste the content of the `main.yaml` file into your existing GitHub Actions workflow.

If you do not have one, you can follow these steps:

1. Copy the content of the exported `main.yaml` file into `.github/workflows/main.yaml` in your project root.
1. To make it fully functional, you may add a `checkout` step before the `Run all kpt functions` step to pull source files from your repo.
1. If you want to see the diff after running kpt functions, append a `Show diff` step.

Your final workflow may looks like this:

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
                args: fn run example-package

          - name: Show diff
            uses: docker://alpine/git
            with:
                args: diff
```

## Viewing the result on GitHub Actions

```shell script
git add .
git commit -am 'Init pipeline'
git push --set-upstream origin master
```

Once the changes are committed and pushed, you can see the latest job on GitHub Actions like this:

![img](/../../../../../static/images/fn-export/github-actions-result.png)

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Once local changes are pushed, you can see how the pipeline fails on GitHub Actions.

[GitHub Actions]: https://github.com/features/actions
[How to Configure a Workflow]: https://docs.github.com/en/actions/configuring-and-managing-workflows/configuring-a-workflow
