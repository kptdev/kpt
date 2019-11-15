## tutorials 2-working-with-local-packages

View fetched package information

### Synopsis

Kpt provides various tools for working with local packages once they are fetched.

  First stage a package to work with

	kpt get  https://github.com/kubernetes/examples/mysql-wordpress-pd wordpress/

## Viewing package structure

	$ kyaml tree wordpress
	wordpress
	├── [gce-volumes.yaml]  v1.PersistentVolume wordpress-pv-1
	├── [gce-volumes.yaml]  v1.PersistentVolume wordpress-pv-2
	├── [local-volumes.yaml]  v1.PersistentVolume local-pv-1
	├── [local-volumes.yaml]  v1.PersistentVolume local-pv-2
	├── [mysql-deployment.yaml]  v1.PersistentVolumeClaim mysql-pv-claim
	├── [mysql-deployment.yaml]  apps/v1.Deployment wordpress-mysql
	├── [mysql-deployment.yaml]  v1.Service wordpress-mysql
	├── [wordpress-deployment.yaml]  apps/v1.Deployment wordpress
	├── [wordpress-deployment.yaml]  v1.Service wordpress
	└── [wordpress-deployment.yaml]  v1.PersistentVolumeClaim wp-pv-claim

  tree summarizes the package Files and Resources

## View the package Resources

	$ kyaml cat wordpress/
	apiVersion: v1
	kind: PersistentVolume
	metadata:
	  name: wordpress-pv-1
	  annotations:
		io.kpt.dev/mode: 420
		io.kpt.dev/package: .
		io.kpt.dev/path: gce-volumes.yaml
	spec:
	  accessModes:
	  - ReadWriteOnce
	  capacity:
		storage: 20Gi
	  gcePersistentDisk:
		fsType: ext4
		pdName: wordpress-1
	---
	apiVersion: v1
	...

  cat prints the raw package Resources.

## Format the Resources for a package (like go fmt)

	$ kpt fmt wordpress/

  fmt formats the Resource Configuration by applying a consistent ordering of fields
  and indentation.

## Search for local package Resources by field

	$ kyaml grep "metadata.name=wordpress" wordpress/
	apiVersion: v1
	kind: Service
	metadata:
	  name: wordpress
	  labels:
		app: wordpress
	  annotations:
		io.kpt.dev/mode: 420
		io.kpt.dev/package: .
		io.kpt.dev/path: wordpress-deployment.yaml
	spec:
	  ports:
	  - port: 80
	  selector:
		app: wordpress
		tier: frontend
	  type: LoadBalancer
	---
	...

  grep prints Resources matching some field value.  The Resources are annotated with their
  file source so they can be piped to other commands without losing this information.

	$ kyaml grep "spec.status.spec.containers[name=nginx].image=mysql:5\.6" wordpress/
	apiVersion: apps/v1 # for k8s versions before 1.9.0 use apps/v1beta2  and before 1.8.0 use extensions/v1beta1
	kind: Deployment
	metadata:
	  name: wordpress-mysql
	  labels:
		app: wordpress
	spec:
	  selector:
		matchLabels:
		  app: wordpress
		  tier: mysql
	  template:
		metadata:
		  labels:
			app: wordpress
			tier: mysql
	...

  - list elements may be indexed by a field value using list[field=value]
  - '.' as part of a key or value may be escaped as '\.'

## Combine grep and tree

	$ kyaml grep "metadata.name=wordpress" wordpress/ | kyaml tree
	.
	├── [wordpress-deployment.yaml]  apps/v1.Deployment wordpress
	└── [wordpress-deployment.yaml]  v1.Service wordpress

  tree will read from stdin if no arguments are provided.  grep can be used with
  tree to only print a subset of the package.

	# display workloads less than 3 replicas
	kyaml grep "spec.template.spec.containers[name=\.*].name=\.*" ./ | kyaml grep "spec.replicas<3" | kyaml tree --replicas

	# display workloads without an image tag
	kyaml grep "spec.template.spec.containers[name=\.*].name=\.*" ./ |  kyaml grep "spec.template.spec.containers[name=\.*].image=\.*:\.*" -v | kyaml tree --image --name

	# display workloads with greater than 1.0 cpu-limits
	kyaml grep "spec.template.spec.containers[name=\.*].resources.limits.cpu>1.0" ./ | kyaml tree --name --resources

## Combing grep and get

	$ kyaml grep "metadata.name=wordpress" wordpress/ | kpt get - ./new-wordpress

  get will create a new package from the Resource Config emitted by grep

	$ kyaml tree new-wordpress/
	new-wordpress
	├── [wordpress_deployment.yaml]  apps/v1.Deployment wordpress
	└── [wordpress_service.yaml]  v1.Service wordpress

## Combine cat and get

	$ kyaml cat pkg/ | my-custom-transformer | kpt get - pkg/

'cat' may be used with 'get' to perform transformations with unit pipes


```
tutorials 2-working-with-local-packages [flags]
```
