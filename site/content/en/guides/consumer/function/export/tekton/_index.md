---
title: 'Exporting a Tekton Pipeline'
linkTitle: 'Tekton'
type: docs
description: >
  Export a Tekton pipeline that runs kpt functions
---

In this tutorial, you will pull an example blueprint that declares Kubernetes resources and two kpt functions. Then you will export a pipeline that runs the functions against the resources on [Tekton](https://tekton.dev/) and modify it to make it fully functional. How to setting up Tekton is also included if you don't have one running yet. This tutorial takes about 20 minutes.

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

## Setting up Tekton on GCP

Follow the instructions in the [Getting Started](https://tekton.dev/docs/getting-started/) guide of Tekton.

1.  [Create a Kubernetes cluster](https://cloud.google.com/kubernetes-engine/docs/quickstart) of version 1.15 or higher on Google Cloud.

    ```shell script
    gcloud container clusters create tekton-cluster --cluster-version=1.15.11-gke.15
    ```

1.  Install Tekton to the cluster.

    ```shell script
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
    ```

1.  Verify every component listed in the following command has the status `Running`.

    ```shell script
    kubectl get pods --namespace tekton-pipelines
    ```

To make the exported pipeline fully functional, you probably need to do the following steps

1.  Install [Git Tasks](https://github.com/tektoncd/catalog/tree/v1beta1/git) from Tekton Catalog.

    ```shell script
    kpt pkg get https://github.com/tektoncd/catalog/git@v1beta1 git
    kubectl apply -f git/git-clone.yaml
    ```

1.  Provide a Persistent Volume for storage purposes.

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
kpt fn export \
    resources \
    --fn-path functions.yaml \
    --workflow tekton \
    --output pipeline.yaml
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
          - $(workspaces.source.path)/resources
          - --fn-path
          - $(workspaces.source.path)/functions.yaml
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
---
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

If you do not have one, you can copy the exported `pipeline.yaml` into your project root. To make it fully functional, you may need to add a `fetch-repository` as the first task in the pipeline.This task clones your github repo to the Tekton workspace. And make sure `run-kpt-functions` runs after it.

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
          - $(workspaces.source.path)/resources
          - --fn-path
          - $(workspaces.source.path)/functions.yaml
        volumeMounts:
          - name: docker-socket
            mountPath: /var/run/docker.sock
    volumes:
      - name: docker-socket
        hostPath:
            path: /var/run/docker.sock
            type: Socket
---
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

To start the pipeline, run:

```shell script
kubectl apply -f pipeline.yaml
tkn pipeline start run-kpt-functions
```

In the prompt, enter `shared-workspace` as workspace name, leave `Value of the Sub Path` blank, select `pvc` as `Type of the Workspace`, enter `workspace-pvc` as `Value of Claim Name`.

{{< png src="images/fn-export/tekton-result" >}}

To view the output, run

```shell script
tkn pipeline logs
```
