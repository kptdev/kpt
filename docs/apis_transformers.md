## apis transformers



### Synopsis

Description:
  Transformers are the client-side version of Controllers (which implement Kubernetes APIs on the
  server-side).  The are equivalent to Kustomize Transformer plugins.

  Transformers are identified as a Resource that either:
  - have an apiVersion starting with *.gcr.io or docker.io
  - have an annotation 'kpt.dev/container: CONTAINER_NAME'

  When 'kpt reconcile pkg/' is run, it will run instances of containers it finds from
  Transformers, passing in both the Transformer Resource and the full set of Resources in
  the package to the container via stdin.  The Transformer writes out
  the new set of package resources to stdout, and these are written back to the package.

  Transformers may be used to:
  - Generate new Resources from abstractions
  - Apply cross-cutting values to all Resource in the package
  - Enforce cross-cutting policy constraints amongst all Resource in the package

  Examples of Transformers:
  - Replace a field on some / all Resources from the transformer config or an environment variable
  - Define abstractions and generate Resources using templates, DSLs, typescript programs, etc
  - Validate all container images use a tag
  - Validate all workloads have a PodDisruptionBudget

  kpt will pass the config and resources to stdin using an InputOutputList:

	apiVersion: kpt.dev/v1alpha1
	kind: InputOutputList
	functionConfig:
	  the: transformer-resource
	  read:
	    from:
	      - the
	      - package
	items:
	- apiVersion: apps/v1
	  kind: Deployment
	  spec:
	    template: {}
	- apiVersion: v1
	  kind: Service
    
  The Transformer will write the new configs to stdout as an InputOutputList.

  Transformers may:
  - pipe their output to 'kpt fmt --set-filenames' to set filenames on the outputs
  - pipe their output to 'kpt merge' to merge multiple copies of the same Resource,
    this is useful to generate new Resources from templates and merge the changes back
    in a non-destructive manner.

  The container is run with the following security restrictions:
  - network is disabled
  - run as 'nobody' user
  - disable privilege escalation
  - container fs is read-only
  - container is deleted after it completes

  See https://github.com/GoogleContainerTools/kpt/testutil/transformer for an Transformer example.


```
apis transformers [flags]
```

### Options

```
  -h, --help   help for transformers
```

### SEE ALSO

* [apis](apis.md)	 - Contains api information for kpt

