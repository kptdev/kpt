---
title: 'Exporting a GitLab CI Pipeline'
linkTitle: 'GitLab CI'
type: docs
description: >
  Export a GitLab CI config file that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [GitLab CI](https://docs.gitlab.com/ee/ci/). This tutorial takes about 5 minutes.

## Before you begin

Before diving into the following tutorial, you may need to create a public repo on GitLab if you don't have one yet, e.g. `function-export-example`.

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
git remote add origin https://gitlab.com/<USER>/<REPO>.git
```

Then you will get an `exmaple-package` directory:

- `resources/resources.yaml`: declares a `Deployment` and a `Namespace`.
- `resources/constraints/`: declares constraints used by the `gatekeeper-validate` function.
- `functions.yaml`: runs two functions from [Kpt Functions Catalog](../../catalog) declaratively:
  - `gatekeeper-validate` enforces constraints over all resources.
  - `label-namespace` adds a label to all Namespaces.

## Exporting a pipeline

```shell script
kpt fn export exmaple-package --workflow gitlab-ci --output .gitlab-ci.yml
```

Running this command will get a .gitlab-ci.yml like this:

```yaml
stages:
  - run-kpt-functions
kpt:
    stage: run-kpt-functions
    image: docker
    services:
      - docker:dind
    script: docker run -v $PWD:/app -v /var/run/docker.sock:/var/run/docker.sock gcr.io/kpt-dev/kpt:latest
        fn run /app/exmaple-package
```

## Integrating with your existing pipeline

Now you can manually copy and paste the `kpt` field in the `.gitlab-ci.yml` file into your existing GitLab CI config file, and merge the `stages` field.

If you don’t have one yet, you can simply copy and paste the file to the root of your repo. It is fully functional.

## Viewing the result on GitLab

Once the changes are committed and pushed to GitLab, you can see the latest jon on GitLab CI like this:

{{< png src="images/fn-export/gitlab-ci-result" >}}
