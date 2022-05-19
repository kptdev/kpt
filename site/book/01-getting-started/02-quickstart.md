In this example, you are going to configure and deploy Nginx to a Kubernetes
cluster.

## Fetch the package

kpt is fully integrated with Git and enables forking, rebasing and versioning a
package of configuration using the underlying Git version control system.

First, let's fetch the _kpt package_ from Git to your local filesystem:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/nginx@v0.9
```

Subsequent commands are run from the `nginx` directory:

```shell
$ cd nginx
```

`kpt pkg` commands provide the functionality for working with packages on Git
and on your local filesystem.

Next, let's quickly view the content of the package:

```shell
$ kpt pkg tree
Package "nginx"
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc
```

As you can see, this package contains 3 resources in 3 files. There is a special
file named `Kptfile` which is used by the kpt tool itself and is not deployed to
the cluster. Later chapters will explain the `Kptfile` in detail.

Initialize a local Git repo and commit the forked copy of the package:

```shell
$ git init; git add .; git commit -m "Pristine nginx package"
```

## Customize the package

At this point, you typically want to customize the package. With kpt, you can
use different approaches depending on your use case.

### Manual Editing

You may want to manually edit the files. For example, modify the value of
`spec.replicas` in `deployment.yaml` using your favorite editor:

```shell
$ vim deployment.yaml
```

### Automating One-time Edits with Functions

The `kpt fn` set of commands enable you to execute programs called _kpt functions_. These
programs are packaged as containers and take in YAML files, mutate or validate them, and then
output YAML.

For instance, you can use a function (`gcr.io/kpt-fn/search-replace:v0.1`) to search and replace all
the occurrences of the `app` key in the `spec` section of the YAML document (`spec.**.app`) and
set the value to `my-nginx`. 

You can use the `kpt fn eval` command to run this mutation on your local files a single time:

```shell
$ kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- by-path='spec.**.app' put-value=my-nginx
```

To see what changes were made to the local package:

```shell
$ git diff
```

### Declaratively Defining Edits

For operations that need to be performed repeatedly, there is a _declarative_ way to define a
pipeline of functions as part of the package (in the `Kptfile`). In this `nginx` package, the author 
has already declared a function (`kubeval`) that validates the resources 
using their OpenAPI schema.

```yaml
pipeline:
  validators:
    - image: gcr.io/kpt-fn/kubeval:v0.3
```

You might want to label all resources in the package. To achieve that, you can
declare `set-labels` function in the `pipeline` section of `Kptfile`. Add this by running the following
command:

```shell
cat >> Kptfile <<EOF
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        env: dev
EOF
```

This function will ensure that the label `env: dev` is added to all the
resources in the package.

The pipeline is executed using the `render` command:

```shell
$ kpt fn render
```

Regardless of how you choose to customize the package — whether by
manually editing it or running one-time functions using `kpt fn eval` — you need to _render_ the
package before applying it the cluster. This ensures all the functions declared
in the package are executed, and the package is ready to be applied to the
cluster.

## Apply the Package

`kpt live` commands provide the functionality for deploying packages to a
Kubernetes cluster.


First, initialize the kpt package:

```shell
$ kpt live init
```

This adds metadata to the `Kptfile` required to keep track of changes made
to the state of the cluster. This 
allows kpt to group resources so that they can be applied, updated, pruned, and
deleted together.

Apply the resources to the cluster:

```shell
$ kpt live apply --reconcile-timeout=15m
```

This waits for the resources to be reconciled on the cluster by monitoring their
status.

## Update the package

At some point, there will be a new version of the upstream `nginx` package, and
you want to merge the upstream changes with changes to your local package.

First, commit your local changes:

```shell
$ git add .; git commit -m "My customizations"
```

Then update to version `v0.10`:

```shell
$ kpt pkg update @v0.10
```

This merges the upstream changes with your local changes using a schema-aware
merge strategy.

Apply the updated resources to the cluster:

```shell
$ kpt live apply --reconcile-timeout=15m
```

## Clean up

Delete the package from the cluster:

```shell
$ kpt live destroy
```

Congrats! You should now have a rough idea of what kpt is and what you can do
with it. Now, let's delve into the details.
