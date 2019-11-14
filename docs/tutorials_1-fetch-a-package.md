## tutorials 1-fetch-a-package

Tutorial for fetching a package from a remote source

### Synopsis

Packages are directories of Kubernetes Resource Configuration which
may be fetched from sources such as git:

- No additional package metadata or structure is required
- Natively compatible with 'kubectl apply' and 'kustomize'
- May be fetched and updated to specific revisions (using git tags or branches).
- May contain non-configuration files or metadata as part of the package

## Fetch a remote package

### Fetching cassandra

  Fetch a "raw" package (e.g. config only -- no kpt metadata) from the kubernetes examples repo.

	kpt get  https://github.com/kubernetes/examples/cassandra cassandra/

  'kpt get' fetched the remote package from HEAD of the
  https://github.com/kubernetes/examples master branch.

	$ kyaml tree cassandra/
	cassandra
	├── [cassandra-service.yaml]  v1.Service cassandra
	├── [cassandra-statefulset.yaml]  apps/v1.StatefulSet cassandra
	└── [cassandra-statefulset.yaml]  storage.k8s.io/v1.StorageClass fast
	
  'kyaml tree' printed the package structure -- displaying both the Resources as well as the
  files the Resources are specified in.

	$ kpt desc cassandra
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| LOCAL DIRECTORY |   NAME    |           SOURCE REPOSITORY            |  SUBPATH  | VERSION | COMMIT  |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| cassandra       | cassandra | https://github.com/kubernetes/examples | cassandra | master  | 1543966 |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+

  'kpt desc LOCAL_PACKAGE' prints information about the source of the package -- e.g. 
  the repo, subdirectory, etc.

### Fetch the guestbook package

	$ kpt get https://github.com/kubernetes/examples/guestbook ./my-guestbook-copy

  The guestbook package contains multiple guest book instances in separate
  subdirectories.

	$ kyaml tree my-guestbook-copy/
	my-guestbook-copy
	├── [frontend-deployment.yaml]  apps/v1.Deployment frontend
	├── [frontend-service.yaml]  v1.Service frontend
	├── [redis-master-deployment.yaml]  apps/v1.Deployment redis-master
	├── [redis-master-service.yaml]  v1.Service redis-master
	├── [redis-slave-deployment.yaml]  apps/v1.Deployment redis-slave
	├── [redis-slave-service.yaml]  v1.Service redis-slave
	├── all-in-one
	│   ├── [frontend.yaml]  apps/v1.Deployment frontend
	│   ├── [frontend.yaml]  v1.Service frontend
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment frontend
	│   ├── [guestbook-all-in-one.yaml]  v1.Service frontend
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-master
	│   ├── [guestbook-all-in-one.yaml]  v1.Service redis-master
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-slave
	│   ├── [guestbook-all-in-one.yaml]  v1.Service redis-slave
	│   ├── [redis-slave.yaml]  apps/v1.Deployment redis-slave
	│   └── [redis-slave.yaml]  v1.Service redis-slave
	└── legacy
		├── [frontend-controller.yaml]  v1.ReplicationController frontend
		├── [redis-master-controller.yaml]  v1.ReplicationController redis-master
		└── [redis-slave-controller.yaml]  v1.ReplicationController redis-slave

  The separate guestbook subpackages contain variants of the same guestbook application.
  To fetch only the all-in-one instance, specify that subdirectory as part of the package.

	$ kpt get https://github.com/kubernetes/examples/guestbook/all-in-one ./new-guestbook-copy

  'kpt get' only fetched the all-in-one subpackage.

	$ kyaml tree new-guestbook-copy
	new-guestbook-copy
	├── [frontend.yaml]  apps/v1.Deployment frontend
	├── [frontend.yaml]  v1.Service frontend
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment frontend
	├── [guestbook-all-in-one.yaml]  v1.Service frontend
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-master
	├── [guestbook-all-in-one.yaml]  v1.Service redis-master
	├── [guestbook-all-in-one.yaml]  apps/v1.Deployment redis-slave
	├── [guestbook-all-in-one.yaml]  v1.Service redis-slave
	├── [redis-slave.yaml]  apps/v1.Deployment redis-slave
	└── [redis-slave.yaml]  v1.Service redis-slave

## Fetch a specific version of a package

	kpt get https://github.com/pwittrock/examples/staging/cockroachdb@v1.0.0 cockroachdb/

  Specifying '@version' after the package uri fetched the package at that revision.
  The version may be a git branch, tag or ref.

## Import a package from a Helm chart

	helm fetch stable/redis
	helm template redis-9.* | kpt get - ./redis-9/

  This imported the expanded package Resources from stdin and created a local kpt package.

	$ kyaml tree redis-9/
	redis-9
	├── [release-name-redis-headless_service.yaml]  v1.Service release-name-redis-headless
	├── [release-name-redis-health_configmap.yaml]  v1.ConfigMap release-name-redis-health
	├── [release-name-redis-master_service.yaml]  v1.Service release-name-redis-master
	├── [release-name-redis-master_statefulset.yaml]  apps/v1beta2.StatefulSet release-name-redis-master
	├── [release-name-redis-slave_service.yaml]  v1.Service release-name-redis-slave
	├── [release-name-redis-slave_statefulset.yaml]  apps/v1beta2.StatefulSet release-name-redis-slave
	├── [release-name-redis_configmap.yaml]  v1.ConfigMap release-name-redis
	└── [release-name-redis_secret.yaml]  v1.Secret release-name-redis

  The names of the Resource files may be configured using the --pattern flag.

	helm fetch stable/redis
	helm template redis-9.* | kpt get - ./redis-9/ --pattern '%n.resource.yaml'
	
  This configured the generated resource file names to be RESOURCENAME.resource.yaml
  instead of RESOURCENAME_RESOURCETYPE.yaml
  Multiple Resources with the same name are put into the same file:

	$ kyaml tree redis-9/
	redis-9
	├── [release-name-redis-headless.resource.yaml]  v1.Service release-name-redis-headless
	├── [release-name-redis-health.resource.yaml]  v1.ConfigMap release-name-redis-health
	├── [release-name-redis-master.resource.yaml]  v1.Service release-name-redis-master
	├── [release-name-redis-master.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-master
	├── [release-name-redis-slave.resource.yaml]  v1.Service release-name-redis-slave
	├── [release-name-redis-slave.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-slave
	├── [release-name-redis.resource.yaml]  v1.ConfigMap release-name-redis
	└── [release-name-redis.resource.yaml]  v1.Secret release-name-redis
	
 Run 'kpt help get' for the set of --pattern options

## Expand Kustomized Configuration into a separate package

  Kustomization directories are natively recognized as kpt packages, however they may
  also be expanded into separate packages.

	kpt get https://github.com/kubernetes-sigs/kustomize/examples/wordpress wordpress/
	kustomize build wordpress/ | kpt get - wordpress-expanded/

  This expanded the Kustomization into a new package

	$ kyaml tree wordpress-expanded/
	wordpress-expanded
	├── [demo-mysql-pass_secret.yaml]  v1.Secret demo-mysql-pass
	├── [demo-mysql_deployment.yaml]  apps/v1beta2.Deployment demo-mysql
	├── [demo-mysql_service.yaml]  v1.Service demo-mysql
	├── [demo-wordpress_deployment.yaml]  apps/v1beta2.Deployment demo-wordpress
	└── [demo-wordpress_service.yaml]  v1.Service demo-wordpress


```
tutorials 1-fetch-a-package [flags]
```

### Options

```
  -h, --help   help for 1-fetch-a-package
```

### SEE ALSO

* [tutorials](tutorials.md)	 - Contains tutorials for using kpt

