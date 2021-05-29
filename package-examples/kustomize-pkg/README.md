# kustomize-pkg

This is a simple package that shows how kpt packages can be used instead of remote bases in kustomize.  That allows you to take advantage of kpt's rebase and local, in-place edits.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [kustomize the config](#kustomize-the-config)
4. [Render the kustomization](#render-the-kustomization)
5. [Apply the package](#apply-the-package)

### Fetch the package

Get the example package on to local using `kpt pkg get`. Note that this package is for the example, wihin that package we are also fetch an nginx sub-package which is an alternative to having a remote base. 

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/kustomize-pkg

fetching package /package-examples/kustomize-pkg from https://github.com/GoogleContainerTools/kpt to kustomize-pkg
```


### View the package contents

List the package contents in a tree structure.

```shell
$ kpt pkg tree kustomize-pkg/

PKG: kustomize-pkg
├── [Kptfile]  Kptfile kustomize-pkg
├── PKG: nginx
│   ├── [Kptfile]  Kptfile nginx
│   ├── [deployment.yaml]  Deployment my-nginx
│   └── [svc.yaml]  Service my-nginx-svc
├── dev
│   ├── [kustomization.yaml]  Kustomization 
│   └── [pass-patch.yaml]  Deployment deployment-patch
└── prod
    ├── [kustomization.yaml]  Kustomization 
    └── [pass-patch.yaml]  Deployment deployment-patch
```

### kustomize the config

We recommend that you keep the kustomize instructions to rendering only such as adding a namespace, transforming or applying a patch.  The second you mix kpt packages and remote bases you will be missing out on a big advantage of having a guaranteed stable base.

You can edit the patches and kustomize files we have created in the overlay folders.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../../bases/nginx
patches:
- path: pass-patch.yaml
  target:
    kind: Deployment
commonLabels:
    environ: dev
namePrefix: dev-
```

### Render the kustomization

You can see the final configuration that the patch is going to render. The major difference between this and in place edit is that you can't just go to the file and look at it in isolation, you need to run `kustomize build` and examine the final results.

```shell
$ kustomize build

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
apiVersion: kpt.dev/v1alpha2
kind: Kptfile
metadata:
  labels:
    environ: dev
  name: dev-nginx
pipeline:
  validators:
  - image: gcr.io/kpt-fn/kubeval:v0.1
upstream:
  git:
    directory: package-examples/nginx
    ref: v0.2
    repo: https://github.com/GoogleContainerTools/kpt
  type: git
  updateStrategy: resource-merge
upstreamLock:
  git:
    commit: 4d2aa98b45ddee4b5fa45fbca16f2ff887de9efb
    directory: package-examples/nginx
    ref: package-examples/nginx/v0.2
    repo: https://github.com/GoogleContainerTools/kpt
  type: git
```

### Apply the package

It's possible to use kustomize build and kpt live apply.  