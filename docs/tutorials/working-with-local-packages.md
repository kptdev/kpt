## tutorials 2-working-with-local-packages

kustomize config

### Synopsis

Kpt provides various tools for working with local packages once they are fetched.

  First stage a package to work with

	kpt get  https://github.com/kubernetes/examples/mysql-wordpress-pd wordpress/

## Combing grep and get

	$ kustomize config grep "metadata.name=wordpress" wordpress/ | kpt get - ./new-wordpress

  get will create a new package from the Resource Config emitted by grep

	$ kustomize config tree new-wordpress/
	new-wordpress
	├── [wordpress_deployment.yaml]  apps/v1.Deployment wordpress
	└── [wordpress_service.yaml]  v1.Service wordpress

## Combine cat and get

	$ kustomize config cat pkg/ | my-custom-transformer | kpt get - pkg/

'cat' may be used with 'get' to perform transformations with unit pipes


```
tutorials 2-working-with-local-packages [flags]
```
