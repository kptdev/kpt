Once you have initialized the package, you can deploy it using `live apply`:

```shell
$ kpt live apply wordpress
installing inventory ResourceGroup CRD.
service/wordpress created
service/wordpress-mysql created
deployment.apps/wordpress created
deployment.apps/wordpress-mysql created
persistentvolumeclaim/mysql-pv-claim created
persistentvolumeclaim/wp-pv-claim created
6 resource(s) applied. 6 created, 0 unchanged, 0 configured, 0 failed
0 resource(s) pruned, 0 skipped, 0 failed
```

Alternatively, you can specify whether you want to wait for resources to be
reconciled:

```shell
$ kpt live apply wordpress --reconcile-timeout=2m
```

?> Refer to the [apply command reference][apply-doc] for usage.

## `ResourceGroup` CRD

By default, `live apply` automatically installs the `ResourceGroup` CRD (unless
`--dry-run` is specified) since it needs to create the associated
`ResourceGroup` custom resource. You can also manually install the CRD before
running `live apply`:

```shell
$ kpt live install-resource-group
```

?> Installing this CRD requires sufficient ClusterRole permission, so you may
need to ask your cluster admin to install it for you.

## Server-side vs Client-side apply

By default, `live apply` command uses client-side apply. The updates are
accomplished by calculating and sending a patch from the client. Server-side
apply, which can be enabled with the `--server-side` flag, sends the entire
resource to the server for the update.

## Dry-run

You can uses the `--dry-run` flag to get break down of operations that will be
performed when applying the package.

For example, before applying the `wordpresss` package for the first time, you
would see that 6 resources would be created:

```shell
$ kpt live apply wordpress --dry-run
service/wordpress created (dry-run)
service/wordpress-mysql created (dry-run)
deployment.apps/wordpress created (dry-run)
deployment.apps/wordpress-mysql created (dry-run)
persistentvolumeclaim/mysql-pv-claim created (dry-run)
persistentvolumeclaim/wp-pv-claim created (dry-run)
6 resource(s) applied. 6 created, 0 unchanged, 0 configured, 0 failed (dry-run)
0 resource(s) pruned, 0 skipped, 0 failed (dry-run)
```

When combined with server-side apply, the resources in the package pass through
all the validation steps on the API server.

## Observe the package

After you have deployed the package, you can get its current status at any time:

```shell
$ kpt live status wordpress
deployment.apps/wordpress is InProgress: Available: 0/1
persistentvolumeclaim/wp-pv-claim is InProgress: PVC is not Bound. phase: Pending
service/wordpress-mysql is Current: Service is ready
persistentvolumeclaim/mysql-pv-claim is Current: PVC is Bound
deployment.apps/wordpress-mysql is InProgress: Available: 0/1
service/wordpress is Current: Service is ready
```

?> Refer to the [status command reference][status-doc] for usage.

## Delete the package

To delete all the resources in a package, you can use the `live destroy`
command:

```shell
$ kpt live destroy wordpress
persistentvolumeclaim/wp-pv-claim deleted
persistentvolumeclaim/mysql-pv-claim deleted
deployment.apps/wordpress-mysql deleted
deployment.apps/wordpress deleted
service/wordpress-mysql deleted
service/wordpress deleted
6 resource(s) deleted, 0 skipped
```

?> Refer to the [destroy command reference][destroy-doc] for usage.

[apply-doc]: /reference/cli/live/apply/
[status-doc]: /reference/cli/live/status/
[destroy-doc]: /reference/cli/live/destroy/
