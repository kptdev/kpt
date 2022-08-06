# Namespace provisioning using kpt CLI

In this guide, we will learn how to create a kpt package from scratch using
`kpt CLI`. We will also learn how to enable customization of the package
with minimal manual steps for package consumers.

## What package are we creating ?

Onboarding a new application or a micro-service is a very common task for a
platform team. It involves provisioning a dedicated namespace (and other
associated resources) where all resources that belong to the application reside.
In this guide, we will create a package that will be used for provisioning a namespace.

## Prerequisites

### Repositories

Platform teams will have to setup two repos:
1. Blueprint repo where reusable kpt packages will live.
2. Deployment repo where instances of packages that will be deployed to a
kubernetes cluster will live. Users will create deployable packages from the
packages in the blueprint repo.

We will refer to the repos with environment variables named $BLUEPRINT_REPO and $DEPLOYMENT_REPO.

If you don’t have these two repositories, you can follow the steps below using
Github’s CLI [gh](https://cli.github.com/) to set up the repos, or set them up
via the GitHub GUI.

```shell

# (optional) skip if you've authenticated. 
# Authenticate gh to create a repository.

$ gh auth login

# Create the "blueprint" and "deployment" repos if you don't have them yet

$ gh repo create blueprint
$ gh repo create deployment


# clone and enter the blueprint repo
$ USER=<YOUR GITHUB USERNAME>
$ BLUEPRINT_REPO=git@github.com:${USER}/blueprint.git
$ DEPLOYMENT_REPO=git@github.com:${USER}/deployment.git

$ git clone ${BLUEPRINT_REPO}
$ git clone ${DEPLOYMENT_REPO}

$ cd blueprint

```

### kube-gen.sh

We will be writing kubernetes manifests from scratch, feel free to use whatever
tools (IDE/extensions) you are most comfortable with. We have created this
little script that helps in generating kubernetes manifests just with kubectl.
So ensure that you have kubectl installed and configured to talk to a
kubernetes cluster.

```shell
# quick check is kubectl is working correctly

$ kubectl get pods
No resources found.
```

Run the following bash snippet to create our little k8s manifest generator called
kube-gen.sh.

```shell
$ cat << 'EOF' > kube-gen.sh

#!/usr/bin/env bash
#kube-gen.sh resource-type args
res="${1}"
shift 1
if [[ "${res}" != namespace ]] ; then
  namespace="--namespace=example"
else
  namespace=""
fi
kubectl create "${res}" -o yaml --dry-run=client "${@}" ${namespace} |\
egrep -v "creationTimestamp|status"
EOF
```
  
Follow the steps below to make sure that script can be invoked from the command line.

```shell
# make the script executable
$ chmod a+x kube-gen.sh

# let's make it available in our $PATH
$ sudo mv kube-gen.sh /usr/local/bin

# test the script out
$ kube-gen.sh --help
Create a resource from a file or from stdin.
 JSON and YAML formats are accepted.
....
```

## Steps

### Initialize a package

```shell
# You should be under the `./blueprint` git directory. If not, check the above
# section  "Prerequisites | Repositories"

# create a directory
$ mkdir basens

# let's initialize the package
$ kpt pkg init basens --description "kpt package for provisioning namespace"
writing basens/Kptfile
writing basens/README.md
writing basens/package-context.yaml

# examine the package content
$ kpt pkg tree basens
Package "basens"
├── [Kptfile]  Kptfile tenant
└── [package-context.yaml]  ConfigMap kptfile.kpt.dev
```
  
## Adding Resources

### Namespace

Now that we have the package initialized we are ready to add basic resources for
provisioning a namespace.

```shell
# ensure that we are working in the basens directory
$ cd basens

# create namespace
$ kube-gen.sh namespace example > namespace.yaml

# you should see namespace resource
$ kpt pkg tree
Package "basens"
├── [Kptfile]  Kptfile tenant
├── [namespace.yaml]  Namespace example
└── [package-context.yaml]  ConfigMap kptfile.kpt.dev
```

Before we add more resources to the package, let's configure our package to
ensure that the namespace for new resources in the package is set correctly.
kpt offers a set of common functions as part of [kpt-function-catalog](https://catalog.kpt.dev)
and it has a [set-namespace](https://catalog.kpt.dev/set-namespace) function
that can be used to ensure all resources in a package use the same namespace.

```shell
# You should be under the "./blueprint/basens" directory.
# Make sure you have kpt autocomplete enabled.
# How it works: 
# Reset your brain, assume you do not know how to use `kpt fn eval`, the goal
# is to find and add a "namespace" function.
# Press the keyboard key `tab` or `tab tab` after each flag `--type`,
# `--keywords`, `--image`, `--fn-config` to see available choices, click `tab`
# to autocomplete your choice or to see further options. 

$ kpt fn eval --type mutator --keywords namespace --image set-namespace:v0.4.1 --fn-config package-context.yaml
[RUNNING] "gcr.io/kpt-fn/set-namespace:v0.4.1"
[PASS] "gcr.io/kpt-fn/set-namespace:v0.4.1" in 600ms
  Results:
    [info]: namespace "example" updated to "example", 0 values changed

# let's add `set-namespace` to rendering workflow so that it is invoked whenever
# package is rendered.
$ kpt fn eval -i set-namespace:v0.4.1 --fn-config package-context.yaml --save -t mutator
[RUNNING] "gcr.io/kpt-fn/set-namespace:v0.4.1"
[PASS] "gcr.io/kpt-fn/set-namespace:v0.4.1" in 600ms
  Results:
    [info]: namespace "example" updated to "example", 0 values changed
 Added "gcr.io/kpt-fn/set-namespace:v0.4.1" as mutator in the Kptfile.

# Let's take a look at Kptfile to see if `set-namespace` is added in the
# rendering pipeline.
$ cat Kptfile
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: basens
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: kpt package for provisioning namespace
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-namespace:v0.4.1
      configPath: package-context.yaml

# render the package to ensure we have a working package.
$ kpt fn render
Package "tenant": 
[RUNNING] "gcr.io/kpt-fn/set-namespace:v0.4.1"
[PASS] "gcr.io/kpt-fn/set-namespace:v0.4.1" in 600ms
  Results:
    [info]: namespace "example" updated to "example", 0 values changed
Successfully executed 1 function(s) in 1 package(s).
```

Note: if you are curious about how KRM functions are implemented. Take a look
at [set-namespace code](https://github.com/GoogleContainerTools/kpt-functions-catalog/blob/master/functions/go/set-namespace/transformer/namespace.go)
to get a feel for the implementation.

### Permissions

Cluster roles for administering the workloads (say app-admin) will already be
created with the correct set of permissions. Organizations will have conventions
such as [example.admin@bigco.com](mailto:tenant-name-admins@mycompany.com)
(think [order-service.admin@bigco.com](mailto:order-service-admins@googlegroups.com))
as the group name responsible for administering namespace example. So let’s
create a rolebinding that grants permissions to the workload service account and
the group [example.admin@bigco.com](mailto:tenant-name-admins@mycompany.com) for
managing this tenant.

```shell
# create rolebinding and try out the simple value propagation scenario
$ kube-gen.sh rolebinding app-admin --clusterrole=app-admin --group=example.admin@bigco.com > rolebinding.yaml

$ cat rolebinding.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: app-admin
  namespace: example
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: app-admin
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: example.admin@bigco.com
```

To enable automatic customization of this role binding for an instance of a
namespace package, we can bind the value of package instance name to the group
name in the above role binding resource. We will use the [apply-replacements function](https://catalog.kpt.dev/apply-replacements/v0.1/)
from kpt-function-catalog for binding the values.
Here is an snippet that does that:

```shell
# get the value of package name from configmap in `package-context.yaml`
# and use it to update the name of the entry in subjects section of app-admin
# role binding with a Group kind. Save the config to update-rolebinding.yaml.
$ cat > update-rolebinding.yaml << EOF

apiVersion: fn.kpt.dev/v1alpha1
kind: ApplyReplacements
metadata:
  name: update-rolebinding
  annotations:
    config.kubernetes.io/local-config: "true"
replacements:
- source:
    kind: ConfigMap
    name: kptfile.kpt.dev
    fieldPath: data.name
  targets:
  - select:
      name: app-admin
      kind: RoleBinding
    fieldPaths:
    - subjects.[kind=Group].name
    options:
      delimiter: '.'
      index: 0
EOF
```

Run following commands to add apply-replacements in the package rendering workflow.

```shell
$ kpt fn eval -i apply-replacements:v0.1.1 --fn-config update-rolebinding.yaml --save -t mutator
[RUNNING] "gcr.io/kpt-fn/apply-replacements:v0.1.1"
[PASS] "gcr.io/kpt-fn/apply-replacements:v0.1.1" in 1s
Added "gcr.io/kpt-fn/apply-replacements:v0.1.1" as mutator in the Kptfile.

# ensure our package is being rendered correctly
$ kpt fn render
```

### Quota

Let’s add quota limits for this tenant.

```shell
$ kube-gen.sh quota default --hard=cpu=40,memory=40G > resourcequota.yaml
$ kpt fn render

$ cat resourcequota.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: default
  namespace: example
spec:
  hard:
    cpu: "40"
    memory: 40G
  
```

So with that, we should have our basic namespace provisioning package ready.
Let’s take a look at what we have:

```shell
$ kpt pkg tree
Package "basens"
├── [Kptfile]  Kptfile basens
├── [namespace.yaml]  Namespace example
├── [package-context.yaml]  ConfigMap kptfile.kpt.dev
├── [resourcequota.yaml]  ResourceQuota example/default
├── [rolebinding.yaml]  RoleBinding example/app-admin
└── [update-rolebinding.yaml]  ApplyReplacements update-rolebinding
```

## Publishing the package

Now that we have a basic namespace package in place, let's publish it so that
other users can consume it.

```shell
$ cd .. && git add basens && git commit -am "initial pkg"
$ git push origin main

$ git tag basens/v0 && git push origin basens/v0
```

So, now the package should be available in the `blueprint` repo. Consumers
(application teams or platform team provisioning namespace on behalf of
application team) will now use this published package to create deployable
instances of it. There are different ways to create deployable instances of
this package:

- [Use package orchestration CLI](guides/porch-user-guide.md)
- Use package orchestration UI (coming soon)
- Use kpt CLI without package orchestration as described in the next section.

### Package Consumption Workflow (without package orchestration)

Assuming you are onboarding a new micro-service called backend, let’s go
through the process of creating an instance of the basens package for backend.
You need to do this step in the deployment repo.

```shell
# Redirect yourself to $DEPLOYMENT_REPO, which is created in "Prerequisites – Repositories
$ cd ../deployment

$ kpt pkg get ${BLUEPRINT_REPO}/basens/@v0 backend --for-deployment
Package "backend":
Fetching ${BLUEPRINT_REPO}@v0
From ${BLUEPRINT_REPO}
 * tag               basens/v0  -> FETCH_HEAD
Adding package "basens".
Fetched 1 package(s).

Customizing package for deployment.
[RUNNING] "builtins/gen-pkg-context"
[PASS] "builtins/gen-pkg-context" in 0s
  Results:
    [info]: generated package context

Customized package for deployment.
```

Render the `backend` package so that the package is customized for the `backend`.

```shell

$ kpt fn render backend
Package "backend":
[RUNNING] "gcr.io/kpt-fn/set-namespace:v0.4.1"
[PASS] "gcr.io/kpt-fn/set-namespace:v0.4.1" in 900ms
  Results:
    [info]: namespace "example" updated to "backend", 3 values changed
[RUNNING] "gcr.io/kpt-fn/apply-replacements:v0.1.1"
[PASS] "gcr.io/kpt-fn/apply-replacements:v0.1.1" in 1s

Successfully executed 2 function(s) in 1 package(s).
# Important thing to note in the above output is that the `set-namespace`
# function updated the namespace for the `backend` package instance automatically.

# examine the output of backend package
$ kpt pkg tree backend
Package "backend"
├── [Kptfile]  Kptfile backend
├── [namespace.yaml]  Namespace backend
├── [package-context.yaml]  ConfigMap kptfile.kpt.dev
├── [resourcequota.yaml]  ResourceQuota backend/default
├── [rolebinding.yaml]  RoleBinding backend/app-admin
└── [update-rolebinding.yaml]  ApplyReplacements update-rolebinding

```

So with that we now have a backend service ready to be onboarded and applied to
the cluster. So the first step would be to commit and tag the backend package in
the deployment repo.

```shell
# assuming you are in deployment repo

$ git add backend && git commit -am "initial pkg for deployment"
$ git push origin main

# tag the package
$ git tag backend/v0 main && git push origin backend/v0

```

## Deploy the package in kubernetes cluster

Now let’s deploy the package using kpt live:

```shell
# assuming you are in the deployment directory

$ kpt live init backend
initializing Kptfile inventory info (namespace: backend)...success
  
$ kpt live apply backend
namespace/backend unchanged
namespace/backend reconciled
resourcequota/default created
rolebinding.rbac.authorization.k8s.io/app-admin created
3 resource(s) applied. 2 created, 1 unchanged, 0 configured, 0 failed
resourcequota/default reconcile pending
rolebinding.rbac.authorization.k8s.io/app-admin reconcile pending
resourcequota/default reconciled
rolebinding.rbac.authorization.k8s.io/app-admin reconciled
3 resource(s) reconciled, 0 skipped, 0 failed to reconcile, 0 timed out
```

## References

[Kubernetes RBAC reference documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
