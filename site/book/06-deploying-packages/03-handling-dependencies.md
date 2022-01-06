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
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
statefulset.apps/wordpress-mysql is NotFound: Resource not found
service/wordpress-mysql is NotFound: Resource not found
service/wordpress-mysql is Current: Service is ready
statefulset.apps/wordpress-mysql is InProgress: Ready: 0/1
statefulset.apps/wordpress-mysql is InProgress: Ready: 0/1
statefulset.apps/wordpress-mysql is Current: Partition rollout complete. updated: 1
deployment.apps/wordpress created
service/wordpress created
2 resource(s) applied. 2 created, 0 unchanged, 0 configured, 0 failed
deployment.apps/wordpress is NotFound: Resource not found
service/wordpress is NotFound: Resource not found
service/wordpress is Current: Service is ready
deployment.apps/wordpress is InProgress: Available: 0/1
deployment.apps/wordpress is Current: Deployment is available. Replicas: 1
```

When you delete the package from the cluster, you can see that
resources are deleted in reverse order:
```shell
$ kpt live destroy wordpress-with-dependencies
deployment.apps/wordpress deleted
service/wordpress deleted
2 resource(s) deleted, 0 skipped
statefulset.apps/wordpress-mysql deleted
service/wordpress-mysql deleted
2 resource(s) deleted, 0 skipped
```

See [depends-on] for more information.

[depends-on]:
  /reference/annotations/depends-on
