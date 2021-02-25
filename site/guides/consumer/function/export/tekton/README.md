---
title: 'Exporting a Tekton Pipeline'
linkTitle: 'Tekton'
weight: 6
type: docs
description: >
  Export a Tekton pipeline that runs kpt functions
---

In this tutorial, you will pull an example package that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [Tekton] and modify it to make it fully functional. Setting up Tekton is also covered if you don't have one running yet. This tutorial takes about 20 minutes.

{{% pageinfo color="info" %}}
A kpt version `v0.32.0` or higher is required.
{{% /pageinfo %}}

## Before you begin

*New to Tekton? Here is a [Getting Started]*.

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

## Setting up Tekton on GCP

Follow the instructions in the [Getting Started] guide of Tekton.

1. Check the [prerequisites].
1. [Create a Kubernetes cluster] of version 1.15 or higher on Google Cloud.

    ```shell script
    gcloud container clusters create tekton-cluster --cluster-version=1.15
    ```

1. Install Tekton to the cluster.

    ```shell script
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
    ```

1. Verify every component listed in the following command has the status `Running`.

    ```shell script
    kubectl get pods --namespace tekton-pipelines
    ```

To make the exported pipeline fully functional, you probably need to do the following steps

1. Install [Git Tasks] from Tekton Catalog.

    ```shell script
    kpt pkg get https://github.com/tektoncd/catalog/git@v1beta1 git
    kubectl apply -f git/git-clone.yaml
    ```

1. Provide a Persistent Volume for storage purposes.

    ```shell script
    cat <<EOF | kubectl apply -f -
    kind: PersistentVolumeClaim
    apiVersion: v1
    metadata:
      name: workspace-pvc
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 10Gi
    EOF
    ```

## Exporting a pipeline

```shell script
kpt fn export example-package --workflow tekton --output pipeline.yaml
```

Running this command will get a `pipeline.yaml` like this:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: source
        mountPath: /source
    steps:
      - name: run-kpt-functions
        image: gcr.io/kpt-dev/kpt:latest
        args:
          - fn
          - run
          - $(workspaces.source.path)/example-package
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
----
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: shared-workspace
    tasks:
      - name: kpt
        taskRef:
            name: run-kpt-functions
        workspaces:
          - name: source
            workspace: shared-workspace
```

## Integrating with your existing pipeline

Now you can manually copy and paste the content of the `pipeline.yaml` into your existing pipeline.

If you do not have one, you can copy the exported `pipeline.yaml` into your project root. To make it fully functional, you may need to add a `fetch-repository` as the first task in the pipeline. This task clones your github repo to the Tekton workspace. Make sure `run-kpt-functions` runs after it.

*Remember to update the `https://github.com/<USER>/<REPO>.git` placeholder with your repo in the following pipeline file*.

If you want to see the diff after running kpt functions, append a `show-diff` step in the `run-kpt-functions` Task.

```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: source
        mountPath: /source
    steps:
      - name: run-kpt-functions
        image: gcr.io/kpt-dev/kpt:latest
        args:
          - fn
          - run
          - $(workspaces.source.path)/example-package
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
      - name: show-diff
        image: gcr.io/kpt-dev/kpt:latest
        args:
          - pkg
          - diff
          - $(workspaces.source.path)/example-package
          - --diff-tool
          - git
          - --diff-tool-opts
          - "--no-pager diff"
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
----
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
    name: run-kpt-functions
spec:
    workspaces:
      - name: shared-workspace
    tasks:
      - name: fetch-repository
        taskRef:
          name: git-clone
        workspaces:
          - name: output
            workspace: shared-workspace
        params:
          - name: url
            value: "https://github.com/<USER>/<REPO>.git"
          - name: deleteExisting
            value: "true"
      - name: kpt
        taskRef:
            name: run-kpt-functions
        workspaces:
          - name: source
            workspace: shared-workspace
```

## Run the pipeline via Tekton CLI

```shell script
git add .
git commit -am 'Init pipeline'
git push --set-upstream origin master
```

Once local changes are committed and pushed. Start the pipeline:

```shell script
kubectl apply -f pipeline.yaml
tkn pipeline start run-kpt-functions
```

In the prompt, enter `shared-workspace` as workspace name, leave `Value of the Sub Path` blank, select `pvc` as `Type of the Workspace`, enter `workspace-pvc` as `Value of Claim Name`.

![img](/static/images/fn-export/tekton-result.png)

To view the output, run

```shell script
tkn pipeline logs
```

## Next step

Try to remove the `owner: alice` line in `example-package/resources/resources.yaml`.

Once local changes are pushed, run the pipeline again, then you can see how it fails.

[Tekton]: https://tekton.dev/
[Getting Started]: https://tekton.dev/docs/getting-started/
[prerequisites]: https://tekton.dev/docs/getting-started#prerequisites
[Create a Kubernetes cluster]: https://cloud.google.com/kubernetes-engine/docs/quickstart
[Git Tasks]: https://github.com/tektoncd/catalog/tree/v1beta1/git
