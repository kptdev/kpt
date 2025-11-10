---
title: "Chapter 6: Deploying Packages"
linkTitle: "Chapter 6: Deploying Packages"
description: |
    In this chapter of this book, we are going to cover how you deploy a kpt package to a Kubernetes cluster and how the
    cluster state is managed as the package evolves over time.
toc: true
menu:
  main:
    parent: "Book"
    weight: 60
---

## Introduction

We use `kpt live apply` instead of `kubectl apply` since it provides some critical
functionality not provided by `kubectl apply`: namely *pruning* and *reconcile status*. To
enable this functionality, we need a cluster-side mechanism for grouping and
tracking resources belonging to a package. This cluster-side grouping is
implemented using a custom resource of kind `ResourceGroup`. Otherwise,
`kpt live` and `kubectl` are complementary. For example, you can use
`kubectl get` as you normally would.

### Pruning

`kpt live apply` will automatically delete cluster resources that are no longer
present in the local package. This clean-up functionality is called pruning.

For example, consider a package which has been applied with the following three
resources:

```shell
service-1 (Service)
deployment-1 (Deployment)
config-map-1 (ConfigMap)
```

The package is updated to contain the following resources:

```shell
service-1 (Service)
deployment-1 (Deployment)
config-map-2 (ConfigMap)
```

When the updated package is applied, `config-map-2` is applied to the cluster and
`config-map-1` is automatically deleted from the cluster.

### Reconcile Status

Kubernetes is based on an asynchronous reconciliation model. When you apply a
resource to a cluster, you actually care about two different things:

- Did the resource apply successfully (synchronous)?
- Did the resource reconcile successfully (asynchronous)?

This is referred to as _apply status_ and _reconcile status_ respectively:

![img](/images/status.svg)

The `kpt live apply` command computes the reconcile status. An example of this could
be applying a `Deployment`. Without computing the reconcile status, the operation
would be reported as successful as soon as the resource has been created in the
API server. With reconcile status, `kpt live apply` will wait until the desired
number of pods have been created and become available.

For core kubernetes types, reconcile status is computed using hardcoded rules.
For CRDs, the status computation is based on the recommended
[convention for status fields](../../reference/schema/crd-status-convention/)
that must be followed by custom resource publishers. If CRDs follow
these conventions, `kpt live apply` will correctly compute the reconciliation status.
`kpt` alsohas special rules for computing status for
[Config Connector resources](../../reference/schema/config-connector-status-convention/).

Multiple resources are usually applied together and we want to know
when all of these resources have been successfully reconciled. `kpt live apply` computes
the aggregate status and waits until either they are all reconciled, the timeout
expires, or all the remaining unreconciled resources have reached a state where they
are unlikely to successfully reconcile. An example of the latter for `Deployment`
resources is when the progress deadline is exceeded.

### Dependency ordering

Sometimes resources must be applied in a specific order. For example,
an application might require that a database is available when it starts.
`kpt live` lets users express these constraints on resources, and uses them
to make sure a resource has been successfully applied and reconciled before
any resources that depend on it are applied.

## Initializing a Package for Apply

Before you can apply the package to the cluster, it needs to be initialized
using `kpt live init`. This is a one-time client-side operation that adds metadata
to the `ResourceGroup` CR (by default located in `resourcegroup.yaml` file)
specifying the name, namespace and inventoryID of the `ResourceGroup` resource
`kpt live apply` command will use to store the inventory (list of the resources applied).

Let's initialize the `wordpress` package:

```shell
kpt live init wordpress
initializing "resourcegroup.yaml" inventory info (namespace: default)...success
```

This creates the `ResourceGroup` CR in the `resourcegroup.yaml` file:

```yaml
apiVersion: kpt.dev/v1alpha1
kind: ResourceGroup
metadata:
  name: inventory-74096247
  namespace: default
  labels:
    cli-utils.sigs.k8s.io/inventory-id: 0a32e2c0200b4bd4c19cd3e097086b4648b8902d-1653113657067255815
```

`ResourceGroup` is a namespace-scoped resource. By default, `kpt live init` command
uses heuristics to automatically choose the namespace. In this example, all the
resources in `wordpress` package are in the `default` namespace, so it chooses
`default` for the namespace. Alternatively, you can manually configure the name
and namespace of the `ResourceGroup` resource. Refer to the
[init command reference](../../reference/cli/live/init/) for usage.

{{< warning type=warning >}}
Once a package is applied to the cluster, do not change the `ResourceGroup` CR. Doing so corrupts the association between the package and the inventory in the cluster, possibly leading to unpredictible and destructive operations.
{{< /warning >}}

## Applying a Package

Once you have initialized the package, you can deploy it using `kpt live apply`.

The `wordpress` package requires a `Secret` containing the mysql password.
Let's create that first:

```shell
kubectl create secret generic mysql-pass --from-literal=password=YOUR_PASSWORD
```
{{< warning type=info >}}
You can also declare the `Secret` resource, but make sure it is not committed to Git as part of the package.
{{< /warning >}}

Then deploy the package and wait for the resources to be reconciled:

```shell
kpt live apply wordpress --reconcile-timeout=2m
installing inventory ResourceGroup CRD.
service/wordpress created
service/wordpress-mysql created
deployment.apps/wordpress created
deployment.apps/wordpress-mysql created
persistentvolumeclaim/mysql-pv-claim created
persistentvolumeclaim/wp-pv-claim created
6 resource(s) applied. 6 created, 0 unchanged, 0 configured, 0 failed
service/wordpress reconcile pending
service/wordpress-mysql reconcile pending
deployment.apps/wordpress reconcile pending
deployment.apps/wordpress-mysql reconcile pending
persistentvolumeclaim/mysql-pv-claim reconcile pending
persistentvolumeclaim/wp-pv-claim reconcile pending
service/wordpress reconciled
service/wordpress-mysql reconciled
persistentvolumeclaim/mysql-pv-claim reconciled
persistentvolumeclaim/wp-pv-claim reconciled
deployment.apps/wordpress-mysql reconciled
deployment.apps/wordpress reconciled
6 resource(s) reconciled, 0 skipped, 0 failed to reconcile, 0 timed out
```

Refer to the [apply command reference](../../reference/cli/live/apply/) for usage.

### `ResourceGroup` CRD

By default, `kpt live apply` automatically installs the `ResourceGroup` CRD (unless
`--dry-run` is specified) since it needs to create the associated
`ResourceGroup` custom resource. You can also manually install the CRD before
running `kpt live apply`:

```shell
kpt live install-resource-group
```

{{< warning type=info >}}
Installing this CRD requires sufficient ClusterRole permission, so you may
need to ask your cluster admin to install it for you.
{{< /warning >}}

### Server-side vs Client-side apply

By default, `kpt live apply` command uses client-side apply. The updates are
accomplished by calculating and sending a patch from the client. Server-side
apply, which can be enabled with the `--server-side` flag, sends the entire
resource to the server for the update.

### Dry-run

You can use the `--dry-run` flag to get break down of operations that will be
performed when applying the package.

For example, before applying the `wordpresss` package for the first time, you
would see that 6 resources would be created:

```shell
kpt live apply wordpress --dry-run
Dry-run strategy: client
service/wordpress created
service/wordpress-mysql created
deployment.apps/wordpress created
deployment.apps/wordpress-mysql created
persistentvolumeclaim/mysql-pv-claim created
persistentvolumeclaim/wp-pv-claim created
6 resource(s) applied. 6 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```

When combined with server-side apply, the resources in the package pass through
all the validation steps on the API server.

### Observe the package

After you have deployed the package, you can get its current status at any time:

```shell
kpt live status wordpress
service/wordpress is Current: Service is ready
service/wordpress-mysql is Current: Service is ready
deployment.apps/wordpress is Current: Deployment is available. Replicas: 1
deployment.apps/wordpress-mysql is Current: Deployment is available. Replicas: 1
persistentvolumeclaim/mysql-pv-claim is Current: PVC is Bound
persistentvolumeclaim/wp-pv-claim is Current: PVC is Bound
```

Refer to the [status command reference](../../reference/cli/live/status/) for usage.

### Delete the package

To delete all the resources in a package, you can use the `kpt live destroy`
command:

```shell
kpt live destroy wordpress
persistentvolumeclaim/wp-pv-claim deleted
persistentvolumeclaim/mysql-pv-claim deleted
deployment.apps/wordpress-mysql deleted
deployment.apps/wordpress deleted
service/wordpress-mysql deleted
service/wordpress deleted
6 resource(s) deleted, 0 skipped, 0 failed to delete
persistentvolumeclaim/wp-pv-claim reconcile pending
persistentvolumeclaim/mysql-pv-claim reconcile pending
deployment.apps/wordpress-mysql reconcile pending
deployment.apps/wordpress reconcile pending
service/wordpress-mysql reconcile pending
service/wordpress reconcile pending
deployment.apps/wordpress-mysql reconciled
deployment.apps/wordpress reconciled
service/wordpress-mysql reconciled
service/wordpress reconciled
persistentvolumeclaim/mysql-pv-claim reconciled
persistentvolumeclaim/wp-pv-claim reconciled
6 resource(s) reconciled, 0 skipped, 0 failed to reconcile, 0 timed out
```

Refer to the [destroy command reference](../../reference/cli/live/destroy/) for usage.

## Handling Dependencies

Sometimes resources within a package have dependencies that require
one resource to be applied and reconciled before another resource.
For example, a package that includes both Wordpress and MySQL might
require that the MySQL `StatefulSet` is running before the Wordpress
`Deployment` is started.

In kpt, this is supported by declaring dependencies with the 
`config.kubernetes.io/depends-on` annotation.

Let's take a look at the `wordpress-with-dependencies` package, a modified
version of the `wordpress` package used earlier:

```shell
kpt pkg get https://github.com/kptdev/kpt.git/package-examples/wordpress-with-dependencies@v0.1
```

You can see that the resources belonging to wordpress have
the `depends-on` annotation  referencing the MySQL `StatefulSet`:

```yaml
# wordpress-with-dependencies/deployment/deployment.yaml (Excerpt)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wordpress
  namespace: default
  labels:
    app: wordpress
  annotations:
    config.kubernetes.io/depends-on: apps/namespaces/default/StatefulSet/wordpress-mysql
```

The syntax for the resource references are:
 * For namespaced resources: `<group>/namespaces/<namespace>/<kind>/<name>`
 * For cluster-scoped resources: `<group>/<kind>/<name>`

Before you can deploy the package, you need to initialize it and create a `Secret`
containing the mysql password:

```shell
kpt live init wordpress-with-dependencies
initializing Kptfile inventory info (namespace: default)...success

kubectl create secret generic mysql-pass --from-literal=password=YOUR_PASSWORD
```

You can deploy the package just like other packages. You can see that the MySQL `StatefulSet`
and `Service` are created and reconciled before the Wordpress `Deployment` and `Service` are applied.

```shell
kpt live apply wordpress-with-dependencies --reconcile-timeout=2m
service/wordpress-mysql created
statefulset.apps/wordpress-mysql created
service/wordpress-mysql reconcile pending
statefulset.apps/wordpress-mysql reconcile pending
service/wordpress-mysql reconciled
statefulset.apps/wordpress-mysql reconciled
service/wordpress created
deployment.apps/wordpress created
4 resource(s) applied. 4 created, 0 unchanged, 0 configured, 0 failed
service/wordpress reconcile pending
deployment.apps/wordpress reconcile pending
service/wordpress reconciled
deployment.apps/wordpress reconciled
4 resource(s) reconciled, 0 skipped, 0 failed to reconcile, 0 timed out
```

When you delete the package from the cluster, you can see that
resources are deleted in reverse order:
```shell
kpt live destroy wordpress-with-dependencies
deployment.apps/wordpress deleted
service/wordpress deleted
deployment.apps/wordpress reconciled
service/wordpress reconciled
statefulset.apps/wordpress-mysql deleted
service/wordpress-mysql deleted
4 resource(s) deleted, 0 skipped, 0 failed to delete
statefulset.apps/wordpress-mysql reconcile pending
service/wordpress-mysql reconcile pending
statefulset.apps/wordpress-mysql reconciled
service/wordpress-mysql reconciled
4 resource(s) reconciled, 0 skipped, 0 failed to reconcile, 0 timed out
```

See [depends-on](../../reference/annotations/depends-on/) for more information.
