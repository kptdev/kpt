### Overview

#### What is kpt?

> A Git-native, semantic-aware, extensible client-side tool for 
> packaging, customizing, validating, and applying Kubernetes resources.

#### Problem space

`kpt` is primarily targeted at engineers who are looking to do configuration
and policy enforcement of their kubernetes infrastructure at scale.  `kpt` 
design is aimed to solve the following problems:

1. Quick creation of blueprints from existing configuration without 
introducing a template or domain specific language into your infrastructure 
stack
2. Updating downstream forks of configuration packages with upstream 
changes with minimal effort and repetitive work
3. A command line (imperative) and configuration file (imperative) way to 
customize the configuration and enforce policy
4. Packaging up parts of customization and validation into reusable building
blocks (KRM functions) and sharing them within your enteprise.
5. Applying and pruning a group of resources in a cluster.

#### Tool interoperability

`kpt` works well with other tools that rely on KRM as the data format:

1. You can easily replace remote bases in `kustomize` with kpt packages and 
avoid network dependency.
2. `kpt` works well with [ConfigSync] 





----
[ConfigSync](https://cloud.google.com/kubernetes-engine/docs/add-on/config-sync/overview)