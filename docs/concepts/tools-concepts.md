## tutorials 4-building-solutions

How to build solutions using kpt with other tools from the ecosystem.

Also see [Building Solutions with kpt](../README.md#building-solutions-with-kpt)

### Packaging: `kpt get`, `kpt update`

  Packaging enables fully or partially specified Resource configuration
  + related artifacts to be published and consumed, as well as facilitates
  updating configuration from upstream.

  Example Use Cases:

  - Fetch a *Blueprint* or *Example* and fork or extend it
  - Fetch *Configuration Function* Resources
  - Fetch configuration to be applied directly to a cluster

  - Update a forked *Blueprint* from upstream
  - Update a *Configuration Function* Resource from upstream
  - Update configuration applied to a cluster

  Example:

  Fetch a Blueprint:

    kpt get https://github.com/kubernetes/examples/cassandra cassandra/

  Update a Blueprint to a specific git commit, merging Resource updates with
  local changes to the fork:
  
    kpt update cassandra@322d78b --strategy resource-merge 

### Development: `kustomize build`, `kustomize config run`

  Development of configuration is about developing the configuration which will
  be applied to an apiserver.

  It may involve a number of activities:

  1. Developing Abstractions

     Abstractions allow a higher-level or more specialized API to be defined
     which may generate other Resources. 

     - Templating Resources -- Jinja, YTT, Helm
     - Generating Resources From DSLs --Cue,  Ksonnet, Jsonnet, Terraform
     - Generating Resources Programmatically -- Starlark, TypeScript
  
  2. Developing Blueprint Customizations

     Blueprints allow low-level Resource configuration to be published and
     customized for a specialized case.

     - Change replica counts
     - Change container image
     - Add environment variables

  3. Developing Variant Customizations

     Variants apply customizations for a specific environment.  They
     are useful when the same package needs to be run in several environments,
     but with different opinions. 

     - Dev, Test, Staging, Production
     - us-west, us-east, us-central, asia, europe

  4. Injecting Cross-Cutting elements into Resources

     Injection is useful for applying policies or cross-cutting logic to
     a collection of Resources.  Notably, the injector may be loosely
     coupled from the package being injected.

     - T-Shirt sizing containers based on annotations
     - Injecting side car containers
     - Injecting init containers

  5. Validating Resources

     Validating Resources is important for applying linting or organizational
     opinions.

     - Ensuring resource reservations are specified
     - Ensuring container images are tagged

  How does kpt facilitate these?

  The kpt architecture enables decoupling programs and tools from
  the packaged configuration itself by applying functions (encapsulated in containers)
  to the local configuration.
  That is the packages themselves contain Resource configuration
  rather than code (e.g. templates, DSLs, etc).  The packaged Resources may
  be modified or expanded by external programs, such as `kustomize`.

  `kustomize` is a tool which can be used to develop configuration by:

   - defining customization variants
   - applying functions which may be used for developing abstractions, cross-cutting
     modifications, and validation

  Example Use Cases:

  - Develop variants for test, staging, production versions of config
  - Develop a high-level "App" abstraction API which takes only a few inputs
    and generates a Deployment, Service and ConfigMap
  - Develop an annotation for t-shirt sizing resource reservations, setting cpu
    and memory to values for small, medium, and large
  - Develop validation to ensure container images are always tagged

  Examples:

  See the [example functions](https://github.com/kubernetes-sigs/kustomize/tree/master/functions/examples)

    kustomize config run DIR

  See the [kustomize examples](https://github.com/kubernetes-sigs/kustomize/tree/master/examples)

    kustomize build DIR  

### Actuation: `kubectl apply`, `kustomize status`, `kustomize prune`

  Applying a collection of configuration may involve several steps, and may require
  orchestrating the actuation of several different packages.  The building blocks of
  actuating configuration are:

  1. Apply

     - Take collection of local Resources and send to the cluster
     - Merge locally defined desired state with cluster defined desired state
       (e.g. keep replica count defined by autoscaler in the cluster)

  2. Status

     - Track the status of the changes until they have been fully rolled out
     - Block until the process completes, or fails to make progress for some period of time 
       (e.g. timesout)

  3. Prune

     - Identify Resources that exist in the cluster, but have been deleted locally and delete them
     - Support diff / dry-run

  The kpt architecture facilitates using the Kubernetes project based tooling,
  such as `kubectl` and `kustomize` for actuating configuration changes.

  Example Use Cases:

  1. Apply a package of configuration to a cluster
  2. Wait until it is successful, printing an error on failure
  3. Delete Resources that have deleted from the package since it was last applied

  **Note:** the actuation steps may be performed by automation using a GitOps approach --
  e.g. trigger Google Cloud Build to perform the actuation when PRs are merged into
  release branches.

  Examples:

    # apply non-local Resources -- skips config-functions
    kustomize config cat DIR | kubectl apply -f -
 
    # block on completion of changes
    kustomize status

    # delete Resources removed from the package
    kustomize prune

### Visibility / Inspection: `kustomize config tree`, `kustomize config grep`

  When working with configuration as data, the configuration may become verbose.
  This makes it challenging to quickly understand the state of the system declared
  locally.
  
  Tools such as `kustomize config tree` help parse and visualize packages of configuration.
  They may be used with tools such as `kustomize config grep` to query configuration.
  
  Example Use Cases:
  
  - Display all the Resources in a package
  - Display all Resources in a package containing an untagged container image
  - Display all Resources in a package containing a container without resource reservations

  Examples:

    # display resources, as well as container names and images
    kustomize config tree DIR/ --name --image

    # find Resources named nginx
    kustomize config grep "metadata.name=nginx" my-dir/    

### Putting It All Together

1. Fetch a package of configuration

       kpt get https://github.com/kubernetes/examples/cassandra cassandra/

2. View the package Resources

       kustomize config tree cassandra/ --image --name

3. Customize or Develop the package

      # view "setters" for the package
      kustomize config set cassandra/

       # add configuration functions, then run
       kustomize config run cassandra/

       # or add kustomize variants that use it as a base
       mkdir prod/
       vi prod/Kustomization.yaml

4. Apply the package to a cluster

       kustomize apply cassandra/
