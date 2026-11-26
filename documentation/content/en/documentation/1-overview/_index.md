---
title: "Overview"
weight: -1
description: >
  kpt overview
---

## What is kpt?
kpt (Kubernetes Package Transformation) supports management of Configuration as Data (CaD).

*Configuration as Data* is an approach to management of configuration (incl. configuration of infrastructure, policy,
services, applications, etc.)  which:

* makes configuration data the source of truth, stored separately from the live
  state
* uses a uniform, serializable data model to represent configuration
* separates code that acts on the configuration from the data and from packages / bundles of the data
* abstracts configuration file structure and storage from operations that act upon the configuration data; clients
  manipulating configuration data don’t need to directly interact with storage (git, container images)

This enables machine manipulation of configuration for Kubernetes and any infrastructure represented in the 
[Kubernetes Resource Model (KRM)](https://github.com/kubernetes/design-proposals-archive/blob/main/architecture/resource-management.md).

![img](/images/cad-overview.svg)

### Configuration as data key principles

* secrets should be stored separately, in a secret-focused storage system
* stores a versioned history of configuration changes by change sets to bundles of related configuration data
* relies on uniformity and consistency of the configuration format, including type metadata, to enable pattern-based
  operations on the configuration data, along the lines of duck typing
* separates schemas for the configuration data from the data, and relies on schema information for strongly typed
  operations and to disambiguate data structures and other variations within the model
* decouples abstractions of configuration from collections of configuration data
* represents abstractions of configuration generators as data with schemas, like other configuration data
* finds, filters / queries / selects, and/or validates configuration data that can be operated on by given code
  (functions)
* finds and/or filters / queries / selects code (functions) that can operate on resource types contained within a body
  of configuration data
* actuation (reconciliation of configuration data with live state) is separate from transformation of configuration
  data, and is driven by the declarative data model
* transformations, particularly value propagation, are preferable to wholesale configuration generation except when the
  expansion is dramatic (say, >10x)
* transformation input generation should usually be decoupled from propagation
* deployment context inputs should be taken from well defined “provider context” objects
* identifiers and references should be declarative
* live state should be linked back to sources of truth (configuration)

### Components of the kpt toolchain

The kpt toolchain includes the following components:

- [**kpt CLI**]({{% relref "/documentation/6-references/cli" %}}): The kpt CLI supports package and function operations, and also   deployment, via
  either direct apply or GitOps. By keeping an inventory of deployed resources, kpt enables resource pruning, aggregated
  status and observability, and an improved preview experience.

- [**Function SDK**](https://github.com/kptdev/krm-functions-sdk): Any general-purpose or domain-specific language can be used to create functions to transform
 and/or validate the YAML KRM input/output format, but we provide SDKs to simplify the function authoring process, in 
  [Go]({{% relref "/book/05-developing-functions#developing-in-Go" %}}). 

- [**Function catalog**](https://catalog.kpt.dev): A catalog of off-the-shelf, tested functions. kpt makes
  configuration easy to create and transform, via reusable functions. Because they are expected to be used for in-place
  transformation, the functions need to be idempotent.
