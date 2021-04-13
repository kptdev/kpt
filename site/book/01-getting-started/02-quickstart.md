In this example, you are going to configure and deploy Nginx to a Kubernetes cluster.

## Fetch the package

kpt is fully integrated with Git and enables forking, rebasing and versioning a package of
configuration using the underlying Git version control system.

First, let's fetch the _kpt package_ from Git to your local filesystem:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/nginx@next
$ cd nginx
```

`kpt pkg` commands provide the functionality for working with packages on Git and on your local
filesystem.

Next, let's quickly view the content of the package:

```shell
$ kpt pkg tree
PKG: nginx
├── [Kptfile]  Kptfile nginx
├── [deployment.yaml]  Deployment my-nginx
└── [svc.yaml]  Service my-nginx-svc
```

As you can see, this package contains 3 resources in 3 files. There is a special file named
`Kptfile` which is used by the kpt tool itself and is not deployed to the cluster. Later chapters
will explain the `Kptfile` in detail.

## Customize the package

At this point, you typically want to customize the package. With kpt, you can use different
approaches depending on your use case.

You may want to manually edit the files. For example, modify the value of `spec.replicas`
in the `Deployment` resource using your favorite editor:

```shell
$ vim deployment.yaml
```

Often, you want to automatically mutate and/or validate resources in a package.
`kpt fn` commands enable you to execute programs called _kpt functions_.

For example, you can automatically set a label with key `env` on all the resources in the package:

```shell
$ kpt fn eval --image gcr.io/kpt-fn/set-label:v0.1 -- env=dev
```

`eval` command can be used for one-time _imperative_ operations. For operations that need to be
performed repeatedly, there is a _declarative_ way to define a pipeline of functions as part of the
package (in the `Kptfile`). This pipeline is executed using the `render` command:

```shell
$ kpt fn render
```

In this case, the author of the Nginx package has already declared a function (`kubeval`) that
validates the resources using their OpenAPI schema.

In general, regardless of the how you choose to customize the package — whether by manually editing
it or running imperative functions — you need to _render_ the package before applying it the
cluster. This ensure all the functions declared in the package are executed and the package is ready
to be applied to the cluster.

## Apply the Package

`kpt live` commands provide the functionality for deploying packages to a Kubernetes cluster.

First, initialize the package:

```shell
$ kpt live init
```

This adds some metadata to the `Kptfile` required to keep track of changes made to the state of the
cluster. For example, if a resource is deleted from the package in the future, it will be pruned
from the cluster.

You can preview the changes that will be made to the cluster:

```shell
$ kpt live preview
```

Finally, apply the resources to the cluster:

```shell
$ kpt live apply --reconcile-timeout=15m
```

This waits for the resources to be reconciled on the cluster by monitoring their status.

## Clean up

Delete the package from the cluster:

```shell
$ kpt live destroy
```

Congrats! You should now have a rough idea of what kpt is and what you can do with it.
Now, let's delve into the details.

TODO: Add the output from running all the commands.
TODO: Version the nginx example using a tag.
