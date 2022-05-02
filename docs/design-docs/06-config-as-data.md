# Configuration as Data

* Author(s): Martin Maly, @martinmaly
* Approver: @bgrant0607

## Why

This document provides background context for Package Orchestration, which is
further elaborated in a dedicated [document](07-package-orchestration.md).

## Configuration as Data

*Configuration as Data* is an approach to management of configuration (incl.
configuration of infrastructure, policy, services, applications, etc.) which:

* makes configuration data the source of truth, stored separately from the live
  state
* uses a uniform, serializable data model to represent configuration
* separates code that acts on the configuration from the data and from packages
  / bundles of the data
* abstracts configuration file structure and storage from operations that act
  upon the configuration data; clients manipulating configuration data don’t
  need to directly interact with storage (git, container images)

![CaD Overview](./CaD%20Overview.svg)

## Key Principles

A system based on CaD *should* observe the following key principles:

* secrets should be stored separately, in a secret-focused storage system
  ([example](https://cloud.google.com/secret-manager))
* stores a versioned history of configuration changes by change sets to bundles
  of related configuration data
* relies on uniformity and consistency of the configuration format, including
  type metadata, to enable pattern-based operations on the configuration data,
  along the lines of duck typing
* separates schemas for the configuration data from the data, and relies on
  schema information for strongly typed operations and to disambiguate data
  structures and other variations within the model
* decouples abstractions of configuration from collections of configuration data
* represents abstractions of configuration generators as data with schemas, like
  other configuration data
* finds, filters / queries / selects, and/or validates configuration data that
  can be operated on by given code (functions)
* finds and/or filters / queries / selects code (functions) that can operate on
  resource types contained within a body of configuration data
* *actuation* (reconciliation of configuration data with live state) is separate
  from transformation of configuration data, and is driven by the declarative
  data model
* transformations, particularly value propagation, are preferable to wholesale
  configuration generation except when the expansion is dramatic (say, >10x)
* transformation input generation should usually be decoupled from propagation
* deployment context inputs should be taken from well defined “provider context”
  objects
* identifiers and references should be declarative
* live state should be linked back to sources of truth (configuration)

## KRM CaD

Our implementation of the Configuration as Data approach (
[kpt](https://kpt.dev),
[Config Sync](https://cloud.google.com/anthos-config-management/docs/config-sync-overview),
and [Package Orchestration](https://github.com/GoogleContainerTools/kpt/tree/main/porch))
build on the foundation of
[Kubernetes Resource Model](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md)
(KRM).

**Note**: Even though KRM is not a requirement of Config as Data (just like
Python or Go templates or Jinja are not specifically requirements for
[IaC](https://en.wikipedia.org/wiki/Infrastructure_as_code)), the choice of
another foundational config representation format would necessitate
implementing adapters for all types of infrastructure and applications
configured, including Kubernetes, CRDs, GCP resources and more. Likewise, choice
of another configuration format would require redesign of a number of the
configuration management mechanisms that have already been designed for KRM,
such as 3-way merge, structural merge patch, schema descriptions, resource
metadata, references, status conventions, etc.

**KRM CaD** is therefore a specific approach to implementing *Configuration as
Data* which:
* uses [KRM](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md)
  as the configuration serialization data model
* uses [Kptfile](https://kpt.dev/reference/schema/kptfile/) to store package
  metadata
* uses [ResourceList](https://kpt.dev/reference/schema/resource-list/) as a
  serialized package wire-format
* uses a function `ResourceList → ResultList` (`kpt` function) as the
  foundational, composable unit of package-manipulation code (note that other
  forms of code can manipulate packages as well, i.e. UIs, custom algorithms
  not necessarily packaged and used as kpt functions)

and provides the following basic functionality:

* load a serialized package from a repository (as `ResourceList`) (examples of
  repository may be one or more of: local HDD, Git repository, OCI, Cloud
  Storage, etc.)
* save a serialized package (as `ResourceList`) to a package repository
* evaluate a function on a serialized package (`ResourceList`)
* [render](https://kpt.dev/book/04-using-functions/01-declarative-function-execution)
  a package (evaluate functions declared within the package itself)
* create a new (empty) package
* fork (or clone) an existing package from one package repository (called
  upstream) to another (called downstream)
* delete a package from a repository
* associate a version with the package; guarantee immutability of packages with
  an assigned version
* incorporate changes from the new version of an upstream package into a new
  version of a downstream package
* revert to a prior version of a package

## Value

The Config as Data approach enables some key value which is available in other
configuration management approaches to a lesser extent or is not available
at all.

*CaD* approach enables:

* simplified authoring of configuration using a variety of methods and sources
* WYSIWYG interaction with configuration using a simple data serialization
  formation rather than a code-like format
* layering of interoperable interface surfaces (notably GUI) over declarative
  configuration mechanisms rather than forcing choices between exclusive
  alternatives (exclusively UI/CLI or IaC initially followed by exclusively
  UI/CLI or exclusively IaC)
* the ability to apply UX techniques to simplify configuration authoring and
  viewing
* compared to imperative tools (e.g., UI, CLI) that directly modify the live
  state via APIs, CaD enables versioning, undo, audits of configuration history,
  review/approval, pre-deployment preview, validation, safety checks,
  constraint-based policy enforcement, and disaster recovery
* bulk changes to configuration data in their sources of truth
* injection of configuration to address horizontal concerns
* merging of multiple sources of truth
* state export to reusable blueprints without manual templatization
* cooperative editing of configuration by humans and automation, such as for
  security remediation (which is usually implemented against live-state APIs)
* reusability of configuration transformation code across multiple bodies of
  configuration data containing the same resource types, amortizing the effort
  of writing, testing, documenting the code
* combination of independent configuration transformations
* implementation of config transformations using the languages of choice,
  including both programming and scripting approaches
* reducing the frequency of changes to existing transformation code
* separation of roles between developer and non-developer configuration users
* defragmenting the configuration transformation ecosystem
* admission control and invariant enforcement on sources of truth
* maintaining variants of configuration blueprints without one-size-fits-all
  full struct-constructor-style parameterization and without manually
  constructing and maintaining patches
* drift detection and remediation for most of the desired state via continuous
  reconciliation using apply and/or for specific attributes via targeted
  mutation of the sources of truth

## Related Articles

For more information about Configuration as Data and Kubernetes Resource Model,
visit the following links:

* [Rationale for kpt](https://kpt.dev/guides/rationale)
* [Understanding Configuration as Data](https://cloud.google.com/blog/products/containers-kubernetes/understanding-configuration-as-data-in-kubernetes)
  blog post.
* [Kubernetes Resource Model](https://cloud.google.com/blog/topics/developers-practitioners/build-platform-krm-part-1-whats-platform)
  blog post series
