## apis transformers



### Synopsis

Description:
  Transformers are the client-side version of Controllers (which implement Kubernetes APIs on the
  server-side).  The are equivalent to Kustomize Transformer plugins.

  Transformers are identified as a Resource that either:
  - have an apiVersion starting with *.gcr.io or docker.io
  - have an annotation 'kpt.dev/container: CONTAINER_NAME'

  When 'kpt reconcile pkg/' is run, it will run instances of containers it finds from
  Transformers, passing in both the Transformer Resource to the container (via Env Var)
  and the full set of Resources in the package (via stdin).  The Transformer writes out
  the new set of package resources to its stdout, and these are written back to the package.
  Note: the container has the network disabled (loopback only), so it cannot fetch remote files.

  Transformers may be used to:
  - Generate new Resources from abstractions
  - Apply cross-cutting values to all Resource in the package
  - Enforce cross-cutting policy constraints amongst all Resource in the package

  Examples of Transformers:
  - Replace a field on some / all Resources from the transformer config or an environment variable
  - Define abstractions and generate Resources using templates, DSLs, typescript programs, etc
  - Validate all container images use a tag
  - Validate all workloads have a PodDisruptionBudget

 Transformers may be published as containers whose CMD:
  - Reads the collection of Resources from STDIN
  - Reads the transformer configuration from the API_CONFIG env var.
  - Writes the set of Resources to create or update to STDOUT

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

