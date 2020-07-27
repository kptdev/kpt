---
title: 'Exporting a CircleCI Workflow'
linkTitle: 'CircleCI'
type: docs
description: >
  Export a CircleCI config file that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a workflow that runs the functions against the resources on [CircleCI](https://circleci.com/) and merge it manually to your existing pipeline. This tutorial takes about 10 minutes.

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
    --workflow circleci \
    --output config.yml
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
                   - run: kpt fn run resources --fn-path functions.yaml
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

1.  Copy and paste the `orbs` field
1.  Insert a `checkout` step as the first step in the `run-functions` job.
1.  Add `kpt/run-functions` to your workflow jobs.

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
                  - run: kpt fn run resources --fn-path functions.yaml
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

If you don’t have one yet, you can just copy the exported `config.yml` file into `.circleci/config.yml` in your project root. Then do the steps above to make the pipeline fully functional. Once all changes are pushed into GitHub, you can do the following steps to setting up your project on CircleCI:

1.  Log into [CircleCI](https://circleci.com/) and choose `Log In with GitHub`.
1.  Select an organization if prompted.
1.  Choose your newly created repo and click `Set Up Project`.
1.  Click `Add Manually` since you have already added `.circleci/config.yml`.
1.  Click `Start Building`.

## Viewing the result on CircleCI

Once local changes have been pushed, you can see a latest build running on CircleCI like this:

{{< png src="images/fn-export/circleci-result" >}}
