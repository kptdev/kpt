## Fetch a Package

How to fetch a package from a remote source

### Synopsis

Packages are directories of Configuration published as subdirectories to git repositories.

- No additional package metadata or structure is required for a package to be fetched
- Format is natively compatible with `kubectl apply` and `kustomize`
- May be fetched and updated to specific revisions (using git tags or branches)
- May also include non-configuration files or metadata

### Fetch the Cassandra package

  Fetch a "raw" package (e.g. config only -- no kpt metadata) from the kubernetes examples repo.

	kpt get  https://github.com/kubernetes/examples/cassandra cassandra/

  `kpt get` fetched the remote package from HEAD of the
  https://github.com/kubernetes/examples master branch.

	$ kustomize config tree cassandra/
	cassandra
	├── [cassandra-service.yaml]  v1.Service cassandra
	├── [cassandra-statefulset.yaml]  apps/v1.StatefulSet cassandra
	└── [cassandra-statefulset.yaml]  storage.k8s.io/v1.StorageClass fast
	
  `kustomize config tree` printed the package structure -- displaying both the Resources as well as the
  files the Resources are specified in.

	$ kpt desc cassandra
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| LOCAL DIRECTORY |   NAME    |           SOURCE REPOSITORY            |  SUBPATH  | VERSION | COMMIT  |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+
	| cassandra       | cassandra | https://github.com/kubernetes/examples | cassandra | master  | 1543966 |
	+-----------------+-----------+----------------------------------------+-----------+---------+---------+

  `kpt desc LOCAL_PACKAGE` prints information about the source of the package -- e.g. 
  the repo, subdirectory, etc.

### Fetch the Guestbook package

	$ kpt get https://github.com/kubernetes/examples/guestbook ./my-guestbook-copy

  The guestbook package contains multiple guest book instances in separate
  subdirectories.

	$ kustomize config tree my-guestbook-copy/
	my-guestbook-copy
	├── [frontend-deployment.yaml]  apps/v1.Deployment frontend
	├── [frontend-service.yaml]  v1.Service frontend
    ...
	├── all-in-one
	│   ├── [frontend.yaml]  apps/v1.Deployment frontend
	│   ├── [frontend.yaml]  v1.Service frontend
	│   ├── [guestbook-all-in-one.yaml]  apps/v1.Deployment frontend
    ...
	└── legacy
		├── [frontend-controller.yaml]  v1.ReplicationController frontend
		├── [redis-master-controller.yaml]  v1.ReplicationController redis-master
		└── [redis-slave-controller.yaml]  v1.ReplicationController redis-slave

  The separate guestbook subpackages contain variants of the same guestbook application.
  To fetch only the all-in-one instance, specify the instance subdirectory as
  part of the package.

	$ kpt get https://github.com/kubernetes/examples/guestbook/all-in-one ./new-guestbook-copy

  `kpt get` only fetched the all-in-one subpackage.

	$ kustomize config tree new-guestbook-copy
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

### Package Versioning

  Since packages are stored in git, git references may be used to fetch a specific version
  of a package.

	kpt get https://github.com/GoogleContainerTools/kpt/package-examples/hello-world@v0.1.0 hello-world/

  Specifying '@version' after the package uri fetched the package at that revision.
  The version may be a git branch, tag or ref.
  
  Note: git references may also be used with `kpt update` to rollout new configuration versions.
  See `kpt help update` for more information.

### New Package From Kustomize Output

  `kpt get` may also be used to convert `kustomize` output into a package

    # fetch a kustomize example
	kpt get https://github.com/kubernetes-sigs/kustomize/examples/wordpress wordpress/
	
	# build the kustomize package and use `kpt get` to write the output to a directory
	kustomize build wordpress/ | kpt get - wordpress-expanded/

  This expanded the Kustomization into a new package

	$ kustomize config tree wordpress-expanded/
	wordpress-expanded
	├── [demo-mysql-pass_secret.yaml]  v1.Secret demo-mysql-pass
	├── [demo-mysql_deployment.yaml]  apps/v1beta2.Deployment demo-mysql
	├── [demo-mysql_service.yaml]  v1.Service demo-mysql
	├── [demo-wordpress_deployment.yaml]  apps/v1beta2.Deployment demo-wordpress
	└── [demo-wordpress_service.yaml]  v1.Service demo-wordpress

### New Package From Helm Output

  `kpt get` may be used to write expanded `helm` templates to packages.

	helm fetch stable/redis
	helm template redis-9.* | kpt get - ./redis-9/

  This imported the expanded package Resources from stdin and created a local kpt package.

	$ kustomize config tree redis-9/
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

	$ kustomize config tree redis-9/
	redis-9
	├── [release-name-redis-headless.resource.yaml]  v1.Service release-name-redis-headless
	├── [release-name-redis-health.resource.yaml]  v1.ConfigMap release-name-redis-health
	├── [release-name-redis-master.resource.yaml]  v1.Service release-name-redis-master
	├── [release-name-redis-master.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-master
	├── [release-name-redis-slave.resource.yaml]  v1.Service release-name-redis-slave
	├── [release-name-redis-slave.resource.yaml]  apps/v1beta2.StatefulSet release-name-redis-slave
	├── [release-name-redis.resource.yaml]  v1.ConfigMap release-name-redis
	└── [release-name-redis.resource.yaml]  v1.Secret release-name-redis
	
 Run `kpt help get` for more information on --pattern options
