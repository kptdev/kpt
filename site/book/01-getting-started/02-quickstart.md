In this example, you are going to configure and deploy Nginx to a Kubernetes
cluster.

## Fetch the package

kpt is fully integrated with Git and enables forking, rebasing and versioning a
package of configuration using the underlying Git version control system.

First, let's fetch the _kpt package_ from Git to your local filesystem:

{{% hide %}}

<!-- @makeWorkplace @verifyBook-->
```
# Set up workspace for the test.
setupWorkspace

# Create output file.
createOutputFile
```

<!-- @pkgGet @verifyBook-->
```shell
kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/nginx@v0.4
cd nginx
```

{{% /hide %}}

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/nginx@v0.4
$ cd nginx
```

`kpt pkg` commands provide the functionality for working with packages on Git and on your local
filesystem.

Next, let's quickly view the content of the package:

{{% hide %}}

<!-- @pkgTree @verifyBook-->
```shell
kpt pkg tree > output.txt
expectedOutput "Package \"nginx\"
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc"
```

{{% /hide %}}

```shell
$ kpt pkg tree
Package "nginx"
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc
```

As you can see, this package contains 3 resources in 3 files. There is a special file named
`Kptfile` which is used by the kpt tool itself and is not deployed to the cluster. Later chapters
will explain the `Kptfile` in detail.

Initialize a local Git repo and commit the forked copy of the package:

```shell
$ git init; git add .; git commit -m "Pristine nginx package"
```

## Customize the package

At this point, you typically want to customize the package. With kpt, you can
use different approaches depending on your use case.

You may want to manually edit the files. For example, modify the value of
`spec.replicas` in `deployment.yaml` using your favorite editor:

```shell
$ vim deployment.yaml
```

Often, you want to automatically mutate and/or validate resources in a package.
`kpt fn` commands enable you to execute programs called _kpt functions_. For
instance, you can automatically search and replace all the occurrences of `app`
name on resources in the package using path expressions:

{{% hide %}}

<!--@fnEval @verifyBook-->
```shell
kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-path=spec.**.app' 'put-value=my-nginx'
```

{{% /hide %}}

```shell
$ kpt fn eval --image gcr.io/kpt-fn/search-replace:v0.1 -- 'by-path=spec.**.app' 'put-value=my-nginx'
```

To see what changes were made to the local package:

```shell
$ git diff
```

`eval` command can be used for one-time _imperative_ operations. For operations
that need to be performed repeatedly, there is a _declarative_ way to define a
pipeline of functions as part of the package (in the `Kptfile`). For example,
you might want label all resources in the package. To achieve that, you can
declare `set-labels` function in the `pipeline` section of `Kptfile`:

```yaml
pipeline:
  mutators:
    - image: gcr.io/kpt-fn/set-labels:v0.1
      configMap:
        env: dev
```

This function will ensure that the label `env: dev` is added to all the
resources in the package.

The pipeline is executed using the `render` command:

{{% hide %}}

<!--@fnRender @verifyBook-->
```shell
kpt fn render
```

{{% /hide %}}

```shell
$ kpt fn render
```

In this case, the author of the `nginx` package has already declared a function
(`kubeval`) that validates the resources using their OpenAPI schema.

In general, regardless of how you choose to customize the package — whether by
manually editing it or running imperative functions — you need to _render_ the
package before applying it the cluster. This ensures all the functions declared
in the package are executed, and the package is ready to be applied to the
cluster.

## Apply the Package

`kpt live` commands provide the functionality for deploying packages to a
Kubernetes cluster.

First, initialize the package:

```shell
$ kpt live init
```

This adds some metadata to the `Kptfile` required to keep track of changes made
to the state of the cluster. For example, if a resource is deleted from the
package in the future, it will be pruned from the cluster.

You can validate the resources and verify that the expected changes will be made
to the cluster:

```shell
$ kpt live apply --dry-run
```

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

Then update to version `v0.5`:

```shell
$ kpt pkg update @v0.5
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
