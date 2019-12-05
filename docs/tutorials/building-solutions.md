## tutorials 4-building-solutions

How to build solutions using kpt with other tools from the ecosystem

### Synopsis

kpt was developed to solve the problem of **fetching and updating configuration packages**.
Rather than solving all problems related to configuration, kpt was designed to be
composed with other solutions developed within the Kubernetes ecosystem such as the
Kubernetes project based tooling.

kpt focusses on a "configuration as data", rather than a "configuration as code".
In this model configuration is packaged as data objects, rather than as imperative
code containing branches and loops.  Configuration as data allows solutions to be
written in different languages and composed together in ways that follow the
unix philosophy.

The following tutorial covers how to compose kpt with other tools in the ecosystem
to build delivery solutions.

### Configuration Overview

The configuration tooling space can be broken down into a number of categories:

1. **Packaging**

   - Fetch -- get a bundle of Resource configuration
   - Update -- pull in upstream changes to Resource configuration
   - Publish -- publish a bundle of Resource configuration

2. **Development**

   - Abstraction -- substitution, generation, injection, etc
   - Customization -- configuring blueprints, defining variants, etc
   - Validation -- policy enforcement, linting, etc

3. **Actuation**

   - Apply -- apply configuration to a cluster
   - Status -- waiting for changes to be fully rolled out
   - Prune -- deletion of Resources no longer appearing in the config

4. **Visibility** /  **Inspection**

   - Search for Resources within a Package matching a constraint
   - Visualize the relationship between Resources
   
5. **Discovery**

   - Discover new publicly published packages

#### Tools

| Category      | Example Tool           | Example Subcommands                               |
|---------------|------------------------|---------------------------------------------------|
| Packaging     | `kpt`                  | `kpt get`, `kpt update`                           |
| Development   | `kustomize`            | `kustomize build`, `kustomize config run`         |
| Actuation     | `kubectl`, `kustomize` | `kubectl apply`, `kustomize status`               |
| Visibility    | `kustomize`            | `kustomize config grep`, `kustomize config tree`  |
| Discovery     | *unsolved*             |                                                   |

### Packaging: `kpt get`, `kpt update`

  Packaging enables fully or partially specified Resource configuration
  + related artifacts to be published and consumed, as well as facilitates
  updating configuration from upstream.

  Example Use Cases:

  - Fetch a *Blueprint* or *Example* and fork or extend it
  - Fetch a *Configuration Function* Resources
  - Fetch a set of configuration to be applied directly to a cluster

  - Update a forked *Blueprint* from upstream
  - Update a *Configuration Function* Resource from upstream
  - Update configuration applied to a cluster

  Example:

  Fetch a Blueprint:

    kpt get https://github.com/kubernetes/examples/cassandra cassandra/
    
  Update a Blueprint to a specific git commit, merging Resources:
  
    kpt update cassandra@322d78b --strategy resource-merge 

### Development: `kustomize build`, `kustomize config run`

  Development of configuration is about developing the configuration which will
  be applied to an apiserver.

  It may involve a number of activities:
  
  1. Developing Abstractions
     
     - Templating Resources -- Jinja, YTT, Helm
     - Generating Resources From DSLs --Cue,  Ksonnet, Jsonnet, Terraform
     - Generating Resources Programmatically -- Starlark, TypeScript
  
  2. Developing Blueprint Customizations
  
     - Change replica counts
     - Change container image
     - Add environment variables
  
  3. Developing Variant Customizations
  
     - Dev, Test, Staging, Production
     - us-west, us-east, us-central, asia, europe
  
  4. Injecting Cross-Cutting elements into Resources
  
     - T-Shirt sizing containers based on annotations
     - Injecting side car containers
     - Injecting init containers
  
  5. Validating Resources
  
     - Ensuring resource reservations are specified
     - Ensuring container images are tagged

  The kpt architecture facilitates decoupling programs and tools from
  the configuration itself by transforming configuration using configuration
  functions.  That is the packages themselves contain Resource configuration
  rather than code (e.g. templates, DSLs, etc).  The packaged Resources may
  be modified or expanded by external programs, such as kustomize.
  
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

2. Inspect the package

       kustomize config tree cassandra/

3. Customize or Develop the package

       # add configuration functions, then run
       kustomize config run cassandra/
        
       # or add kustomize variants that use it as a base
       mkdir prod/
       vi prod/Kustomization.yaml

4. Apply the package to a cluster

       kustomize config cat cassandra/ | kubectl apply -f -
       kustomize status cassandra/
       kustomize prune cassandra/
        
   or
   
       kustomize build prod/ | kubectl apply -f -
       kustomize build prod/| kustomize status
       kustomize build prod/ | kustomize prune

