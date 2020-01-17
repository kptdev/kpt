## kpt overview

#### Tutorials

- [fetch-a-package](tutorials/01-fetch-a-package.md)
- [update-a-local-package](tutorials/02-update-a-local-package.md)
- [publish-a-package](tutorials/03-publish-a-package.md)

#### Design



#### Building Platforms, Solutions and High-Level Systems with kpt

kpt was developed to solve **configuration packaging** only -- and was designed to be composed
with other tools from the ecosystem in order to build higher-level systems and platforms.

As such, **kpt has a minimal feature set by design to maximize its utility as a building block
for platforms** and ability to be composed with other tools.

This sections provides a high-level overview of some of the ways in which kpt may be composed
with tools from the upstream Kubernetes project to build configuration and delivery solutions.

See [building-solutions](tutorials/building-solutions.md) for a more information on this topic.

1. **Packaging**

   Packaging covers how to bundle configuration for reuse.

   - Fetch -- get a bundle of Resource configuration
   - Update -- pull in upstream changes to Resource configuration
   - Publish -- publish a bundle of Resource configuration

2. **Development**

   Development covers how to create and modify configuration, and includes
   how to incorporate and unify opinions from an arbitrary number of sources.

   - Abstraction -- substitution, generation, injection, etc
   - Customization -- configuring blueprints, defining variants, etc
   - Validation -- policy enforcement, linting, etc

3. **Actuation**

   Actuation covers how to take configuration and actuate it by applying it.

   - Apply -- apply configuration to a cluster
   - Status -- waiting for changes to be fully rolled out
   - Prune -- deletion of Resources no longer appearing in the config

4. **Visibility** /  **Inspection**

   Visibility / Inspection covers how to visualize and understand packaged
   configuration, as well as applied Resources.

   - Search for Resources within a Package, Cluster or set of Clusters.
   - Visualize the relationship between Resources
   - Debug Resources

5. **Discovery**

   Discovery includes how to locate new packages, and examples.

   - Discover new publicly published packages from a market place or the web

#### FAQ

See [faq](tutorials/faq.md)

#### Templates and DSLs

Note: If the use of Templates or DSLs is strongly desired, they may be fully expanded into Resource
configuration to be used as a kpt package.  These artifacts used to generated Resource configuration
may be included in the package as supplements.

#### Env Vars

  COBRA_SILENCE_USAGE
  
    Set to true to silence printing the usage on error

  COBRA_STACK_TRACE_ON_ERRORS

    Set to true to print a stack trace on an error

  KPT_NO_PAGER_HELP

    Set to true to print the help to the console directly instead of through
    a pager (e.g. less)
