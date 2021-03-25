Let's jump to an example that demonstrates a typical kpt workflow and quickly
introduce important concepts and features. Following chapters will cover these concepts in detail.

In this example, you are going to configure and deploy Apache Cassandra to a Kubernetes cluster.

## Fetch the package

First, let's fetch the Cassandra package from Git to your local filesystem:

```shell
$ kpt pkg get https://github.com/GoogleContainerTools/kpt/package-examples/cassandra@next cassandra
```

kpt is fully integrated with Git and enables forking, rebasing and versioning a package of configuration using the underlying Git version control system. Since kpt is Git-native, any existing directory in Git containing Kubernetes resources is also a valid kpt package!

Next, let's quickly view the content of the package:

```shell
$ cd cassandra
$ kpt pkg tree
.
├── [Kptfile]  Kptfile cassandra
├── [service.yaml]  Service cassandra
├── [statefulset.yaml]  StatefulSet cassandra
└── [statefulset.yaml]  StorageClass fast
```

As you can see, this package contains four resources in three files (`statfeulset.yaml` contains 2 resources: `StatefulSet` and `StorageClass`). There is a special file named `Kptfile` which is used by the kpt client itself and is not deployed to the cluster. Later chapters will explain the `Kptfile` in detail.

## Customize the package

At this point, you typically want to customize the package. With kpt, you can use different approaches depending on your use case.

You may want to manually edit the files. For example,
modify the value of `CASSANDRA_CLUSTER_NAME` in the `StatefulSet` resource using your favorite editor:

```shell
$ vim cassandra/statefulset.yaml
```

Alternatively, you may want to automatically customize and validate a package using _functions_. For example, you can set a label with key `env` on all the resources in the package:

```shell
$ kpt fn eval --image gcr.io/kpt-fn/set-label:v0.1 -- env=dev
```

This is an imperative one-time operation. It is also possible to declare a function as part of
pipeline in the `Kptfile`. In this case, the author of a package has already declared a validator function (`kubeval`) that validates the resources using their OpenAPI schema.

Regardless of the how you choose to customize the package, you want to _render_ the package before applying it the cluster:

```shell
$ kpt fn render
```

This runs the declared pipeline of functions and writes the fully rendered configuration in-place.

## Apply the Package

You're now ready to apply the package to the cluster.

Initialize the package:

```shell
$ kpt live init
```

This adds some metadata to the `Kptfile` required to maintain an inventory of previously applied
resources so they can be pruned later.

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
