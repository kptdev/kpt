---
title: 'Exporting a CircleCI Workflow'
linkTitle: 'CircleCI'
weight: 5
type: docs
description: >
  Export a CircleCI config file that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a workflow that runs the functions against the resources on [CircleCI] and merge it manually to your existing pipeline. This tutorial takes about 10 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*Unfamiliar with CircleCI? Read [Getting Started Introduction] first*.

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
kpt fn export example-package --workflow circleci --output config.yml
```

Running this command will get a `config.yml` like this:

```yaml
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gcr.io/kpt-dev/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run example-package
        jobs:
            run-functions:
                executor: kpt-container
                steps:
                  - setup_remote_docker
                  - kpt-fn-run
workflows:
    main:
        jobs:
          - kpt/run-functions
```

## Integrating with your existing pipeline

To merge the exported file with your existing pipeline, you can:

1. Copy and paste the `orbs` field
1. Insert a `checkout` step as the first step in the `run-functions` job.
1. If you want to see the diff after running kpt functions, append a `run: git -no-pager diff` step in the `kpt-fn-run` command.
1. Add `kpt/run-functions` to your workflow jobs.

Your final workflow may looks like this:

```yaml
version: "2.1"
orbs:
    kpt:
        executors:
            kpt-container:
                docker:
                  - image: gcr.io/kpt-dev/kpt:latest
        commands:
            kpt-fn-run:
                steps:
                  - run: kpt fn run example-package
                  - run: git --no-pager diff
        jobs:
            run-functions:
                executor: kpt-container
                steps:
                  - checkout
                  - setup_remote_docker
                  - kpt-fn-run
workflows:
    main:
        jobs:
          - kpt/run-functions
```

If you don’t have one yet, you can do the following steps:

1. Copy the exported `config.yml` file into `.circleci/config.yml` in your project root.
1. Do the steps above to make the pipeline fully functional.

Once all changes are pushed into GitHub, you can do the following steps to setting up your project on CircleCI:

1. Log into [CircleCI] and choose `Log In with GitHub`.
1. Select your own account as an organization if prompted.
1. Choose your newly created repo and click `Set Up Project`.
1. Click `Use Existing Config` since you have already added `.circleci/config.yml`.
1. Click `Start Building`.

## Viewing the result on CircleCI

```shell script
git add .
git commit -am 'Init pipeline'
git push --set-upstream origin master
```

Once local changes have been pushed, you can see a latest build running on CircleCI like this:

{{< png src="images/fn-export/circleci-result" >}}

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Once local changes are pushed, you can see how the pipeline fails on CircleCI.

[CircleCI]: https://circleci.com/
[Getting Started Introduction]: https://circleci.com/docs/2.0/getting-started/#section=getting-started
