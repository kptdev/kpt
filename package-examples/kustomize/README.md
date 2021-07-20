# kustomize

## Overview

This example demonstrates using kpt and kustomize together. Specifically it
uses:

- kpt for packaging (`kpt pkg`)
- kpt for actuation (`kpt live`)
- kustomize for customization (`kustomize build`)

## Steps

Fetch this package:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/kustomize
```

You can view the package hierarchy using the `kpt pkg tree` command or a regular
`tree` command:

```shell
$ tree kustomize

kustomize
├── Kptfile
├── README.md
├── bases
│   └── nginx
│       ├── Kptfile
│       ├── deployment.yaml
│       ├── kustomization.yaml
│       └── svc.yaml
└── overlays
    ├── dev
    │   ├── Kptfile
    │   ├── kustomization.yaml
    │   └── pass-patch.yaml
    └── prod
        ├── Kptfile
        ├── kustomization.yaml
        └── pass-patch.yaml
```

The package hierarchy contains several packages which can be classified into
three categories:

1. `kustomize` is the top-level package.
2. `bases/nginx` this package serves as a local base.
3. `overlays/dev` and `overlays/prod` are overlays which have patches for the
   `nginx` base package.

Having a local base that is a kpt package has several advantages over remote
bases:

1. Consumers of the remote base are able to pull in updates only when they are
   ready to update avoiding surprises.
2. Consumer can do in place edits like adding a file or editing a file without
   having to create a patch for everything.

Note that we have added a `kustomization.yaml` in the `base/nginx` for the
`kustomize build` to be able to get all the resources files. The overlays have
their own kustomization instructions which allow per environment changes.

In order to see what the final configuration looks like you can use use the
familiar `kustomize build`:

```shell
$ kustomize build kustomize/overlays/dev
apiVersion: v1
kind: Service
metadata:
  labels:
    app: nginx
    environ: dev
  name: dev-my-nginx-svc
spec:
  ports:
  - port: 80
  selector:
    app: nginx
    environ: dev
  type: LoadBalancer
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    environ: dev
  name: dev-my-nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
      environ: dev
  template:
    metadata:
      labels:
        app: nginx
        environ: dev
    spec:
      containers:
      - image: nginx:1.14.1
        name: nginx
        ports:
        - containerPort: 80
---
apiVersion: kpt.dev/v1
info:
  description: sample description
kind: Kptfile
metadata:
  labels:
    environ: dev
  name: dev-dev
```

This rendered configuration can be deployed with `kubectl` but you can also take
advantage of `kpt live apply` if you extend your kustomization.yaml to include
the Kptfile. You can learn more about kpt `apply` command in the [deployment
chapter].

In our case we created Kptfiles in the overlay folders and added the Kptfile to
the list of kustomize resources `kustomize/overlays/dev/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../bases/nginx
  - Kptfile
patches:
  - path: pass-patch.yaml
    target:
      kind: Deployment
commonLabels:
  environ: dev
namePrefix: dev-
```

In this solution the overlays are targeting different environments, they can be
on different clusters or in different namespaces. In order to take advantage of
kpt live apply the final `kustomize build` needs to contain the inventory
object.

```shell
$ kpt live init kustomize/overlays/dev
initializing Kptfile inventory info (namespace: default)...success
```

You can then pipe the kustomize build to kpt:

```shell
$ kustomize build kustomize/overlays/dev | kpt live apply -

service/dev-my-nginx-svc created
deployment.apps/dev-my-nginx created
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```

You could also consider changing your hydration logic to `kpt fn render` with an
out of place hydration flag, but that would require a manual migration which is
out of scope for this solution. More about the kpt functions and the pipeline
can be found in the [using functions] chapter of The Kpt Book.

[deployment chapter]: https://kpt.dev/book/06-deploying-packages/
[using functions]: https://kpt.dev/book/04-using-functions/
