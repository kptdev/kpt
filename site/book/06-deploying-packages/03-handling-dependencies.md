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
$ kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/wordpress-with-dependencies@v0.1
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
$ kpt live init wordpress-with-dependencies
initializing Kptfile inventory info (namespace: default)...success

$ kubectl create secret generic mysql-pass --from-literal=password=YOUR_PASSWORD
```

You can deploy the package just like other packages. You can see that the MySQL `StatefulSet`
and `Service` are created and reconciled before the Wordpress `Deployment` and `Service` are applied.

```shell
$ kpt live apply wordpress-with-dependencies --reconcile-timeout=2m
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
$ kpt live destroy wordpress-with-dependencies
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

See [depends-on] for more information.

[depends-on]:
  /reference/annotations/depends-on/
