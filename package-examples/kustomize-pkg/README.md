# kustomize-pkg

This is a simple package that shows how kpt packages can be used instead of remote bases in kustomize.  That allows you to take advantage of kpt's rebase and local, in-place edits.

## Steps

1. [Fetch the package](#fetch-the-package)
2. [View the package contents](#view-the-package-contents)
3. [kustomize the config](#kustomize-the-config)
4. [Render the kustomization](#render-the-kustomization)
5. [Apply the package](#apply-the-package)
6. [Clean up resources](#clean-up-resources)

### Fetch the package

Get the example package on to local using `kpt pkg get`. Note that this package is for this example, wihin that package we are also using the nginx sub-package which is an alternative to having a remote base. 

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/kustomize-pkg

fetching package /package-examples/kustomize-pkg from https://github.com/GoogleContainerTools/kpt to kustomize-pkg
```


### View the package contents

List the package contents in a tree structure.

```shell
$ tree kustomize-pkg

kustomize-pkg
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

### kustomize the config

Kustomize is a great tool for out of place hydration, but we don't recommend that you mix kpt packages and remote bases.  It's best to have a clear delienation what you are using each tools for: kpt for packages and remote resources, kustomize for hydration like adding labels, setting namespaces and overlays.

In the example below what was a remote base is now fetched locally as a kpt package and kustomize is used for hydration.

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

### Render the kustomization

You can see the final configuration that the patch is going to render. The major difference between this and in place edit is that you can't just go to the file and look at it in isolation, you need to run `kustomize build` and examine the final results.

```shell
$ kustomize build kustomize-pkg/overlays/dev

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
info:
  description: sample description
kind: Kptfile
metadata:
  labels:
    environ: dev
  name: dev-dev


### Apply the package

It is possible to use kustomize build and kpt live apply, but it does require passing the inventory information to `kpt live apply` from `kustomize build` output.  It is best to have Kptfiles with inventory information in overlay folders in case the variants of the package are deployed to the same cluster.  Every variant of this application will need to be mapped to it\'s own inventory for pruning.  In case you have two variants that use the same inventory information the consequent deploy might wipe out the previous variant.

```shell
$ kpt live init kustomize-pkg/overlays/dev

initializing Kptfile inventory info (namespace: default)...success
```

You might have noticed that the overlays have Kptfiles and they are added to the kustomization.yaml so the contents are passed all the way through the kustomize build.

Kustomize build will need to be piped to kpt live apply:

```shell
$ kustomize build kustomize-pkg/overlays/dev | kpt live apply - 

service/dev-my-nginx-svc created
deployment.apps/dev-my-nginx created
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```

### Clean up resources

You can use kpt to prune and clean up resources from your build by using `kpt live destroy`.  As long as the inventory information is passed in kpt will know how to clean everything up:

```shell
$ kustomize build kustomize-pkg/overlays/dev | kpt live destroy - 

deployment.apps/dev-my-nginx deleted
service/dev-my-nginx-svc deleted
2 resource(s) deleted, 0 skipped
```