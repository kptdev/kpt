---
title: 'Exporting a GitLab CI Pipeline'
linkTitle: 'GitLab CI'
weight: 2
type: docs
description: >
  Export a GitLab CI config file that runs kpt functions
---

In this tutorial, you will pull an example package that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [GitLab CI]. This tutorial takes about 5 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*New to GitLab CI? Here is [Getting Started with GitLab CI]*

Before diving into the following tutorial, you need to create a public repo on GitLab if you don't have one yet, e.g. `function-export-example`.

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

## Exporting a pipeline

```shell script
kpt fn export example-package --workflow gitlab-ci --output .gitlab-ci.yml
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
        fn run /app/example-package
```

## Integrating with your existing pipeline

Now you can manually copy and paste the `kpt` field in the `.gitlab-ci.yml` file into your existing GitLab CI config file, and merge the `stages` field.

If you don’t have one yet, you can simply copy and paste the file to the root of your repo. It is fully functional.

If you want to see the diff after running kpt functions, append an `after_script` field to run `kpt pkg diff`. Your final `.gitlab-ci.yaml` file looks like this:

```yaml
stages:
  - run-kpt-functions
kpt:
    stage: run-kpt-functions
    image: docker
    services:
      - docker:dind
    script: docker run -v $PWD:/app -v /var/run/docker.sock:/var/run/docker.sock gcr.io/kpt-dev/kpt:latest
        fn run /app/example-package
    after_script:
      - docker run -v $PWD:/app gcr.io/kpt-dev/kpt:latest
        pkg diff /app/example-package
        --diff-tool git --diff-tool-opts "--no-pager diff"
```

## Viewing the result on GitLab

```shell script
git add .
git commit -am 'Init pipeline'
git push --set-upstream origin master
```

Once the changes are committed and pushed to GitLab, you can see the latest job on GitLab CI like this:

![img](/../../../../../static/images/fn-export/gitlab-ci-result.png)

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Once local changes are pushed, you can see how the pipeline fails on GitLab CI.

[GitLab CI]: https://docs.gitlab.com/ee/ci/
[Getting Started with GitLab CI]: https://docs.gitlab.com/ee/ci/quick_start/README.html
