---
title: "Architecture"
linkTitle: "Architecture"
weight: 1
type: docs
description: >
   Kpt Architecture
---

## Background

The kpt's design mirrors the Kubernetes control-plane architecture -- with multiple programs
(e.g. controllers) reading and writing shared configuration data (i.e. resources).

### Influences

#### Unix philosophy

Packages of configuration should be small, modular, and reusable.

- [link](https://en.wikipedia.org/wiki/Unix_philosophy)

#### Resource / controller model

Kpt provides commands to fetch, update, modify, and apply configuration. This allows users to reuse and compose various packages of Kubernetes resources.

- [link](https://kubernetes.io/docs/concepts/architecture/controller/)

#### GitOps

GitOps refers to using a version control system as the source of truth for configuration.

- [link](https://www.weave.works/technologies/gitops/)

#### Configuration as data

Many configuration tools conflate data with the operations on that
data (e.g. YAML files embedding a templating language).
As configuration becomes complex, it becomes hard to read and understand.
Our design philosophy is to keep configuration as data in order to manage this complexity.
We do this by keeping resources serialized as either JSON or YAML configuration.

- [link](https://changelog.com/gotime/114#t=00:12:09.18)

#### kpt vs kubernetes

- While the Kubernetes control-plane reads / writes configuration data stored by the apiserver
  (e.g. in etcd or some other database), kpt reads / writes configuration data stored as local
  files (as well as supporting other sources / destinations -- http, git, etc).
- While in the Kubernetes control-plane controllers are implicitly triggered through watches,
  in kpt programs are explicitly triggered through invoking kpt (although this can be done by
  automation).

Kpt provides a hybrid GitOps + resource-controller model.  It's architecture is designed to
enable **composing loosely coupled solutions** through reading and writing to a shared data model
(e.g. resources, controllers, OpenAPI).

## Design Principles

### Configuration-as-Data vs Configuration-as-Code

kpt packages configuration as data (APIs) rather than as code (templates /  DSLs). In this model,
the configuration is static data, which many tools may read & write.

This model has its roots in the Unix Philosophy:

> Expect the output of every program to become the input to another, as yet unknown, program

As well as in the Kubernetes control-plane where resources are read and written by loosely coupled
controllers.

### Shift-Left & GitOps

By enabling resource-controller style systems to be built on configuration files before
applying them to the cluster, kpt shifts the control-plane left -- enabling more issues to be
caught before they are pushed into a cluster.

- Cluster changes may be reviewed, approved, validated, audited and rolled back using git
- Git automation may be applied to changes to cluster state

### Shift-Right vs Shift-Left

- Read resources from apiserver => Read resources from files
- Write resources to apiserver => Write resources to files
- Triggered by *watch* => triggered by `sh`

### Composability

Design solutions to work together, expecting the output of each command to be read as input
by another.

- Commands read configuration and write configuration.
- The inputs and outputs should be symmetric.
- It should be possible to pipe kpt commands together, and it should be possible to write
  the output of a kpt command back to its source (updating it in-place).

Tools which read / write data may be developed in different languages and composed together.
Tools which write configuration back to the same source it was read from should retain comments
set on the input, as the comments may be used by kpt and other tools as metadata.

Programs may be developed independently of one another, with commands built directly into
the CLI -- e.g. `kpt cfg set`.

Additionally, kpt offers *functions* as an extension mechanism to simplify publishing logic,
and to provide deeper integration with kpt -- e.g. invoking functions automatically
after kpt commands.

#### kpt vs kubernetes

- kpt: resource configuration is read, modified and written back to its source (or another destination)
  - resources may be updated using 3-way merge (kpt pkg update)
- kubernetes: resources are read, modified, and written back to the apiserver
  - resources may be updated using 3-way merge (kubectl apply)

### Resource oriented

Desired system state is expressed using Kubernetes resources -- declarative, static data structures.
The desired state is changed through modification of resources -- these may be done:

- programmatically by manually invoked tools -- e.g. `kpt cfg set`
- through direct text edits -- e.g. `vi`
- through forms of automation -- e.g. GitHub actions

High-level logic should be built into programs which understand configuration and are capable of
generating and transforming it given some context.

Since all tools read and write configuration data, multiple tools may be composed by
invoking them against the same configuration data and pipelining their commands.

#### kpt vs kubernetes

- kpt:
  - read / write files, http, ...
  - triggered explicitly (kpt invocations)
- kubernetes:
  - read / write http
  - triggered implicitly (watches)

### Schema driven

Type or object specific logic should NOT be built into the tool.
Static resource modifications (e.g. *set*) should be configured using type or object
metadata (e.g. schema).

OpenAPI is used for resource schema.  Tools may support their own OpenAPI
extensions which should co-exist with extensions owned by other tools.

Support for new types should be introduced through new OpenAPI definitions rather than
changes to the tool itself.

- Static per-type and per-object resource transformations should use OpenAPI to tell the
  tool how to modify a given object
  - e.g. where to set `image` for `set image` is defined in OpenAPI rather than hard-coded
  
- Configuration for individual objects / resources may define custom OpenAPI definitions for
  that specific instance
  - e.g. an nginx Deployment's `image` may be restricted to the regular expression `^nginx:.*$`

#### kpt vs kubernetes

- kpt: OpenAPI read from multiple sources -- can also be inlined into individual
  configuration objects as comments
- kubernetes: OpenAPI read from apiserver

### Layering

High-level layers should exist to reduce inherent complexity and simplify simple cases.
Lower-level layers should remain accessible, but in the background.

Example:  When high-level solutions generate lower-level resources, those resources
should be accessible to other tools to read and modify them.  If an
nginx abstraction generates a Deployment and Service, it should be possible
for other tools to observe and modify both.

## IO

Both kpt inputs and outputs should be recognized by Kubernetes project tools, published as
Kubernetes examples or published by the Kubernetes apiserver.

Examples:

- `kubectl get -o yaml`
- `kubectl apply -f -`
- `github.com/kubernetes/examples`

Much like Kubernetes controllers, kpt should be able to read its previous outputs and modify
them, rather than generating them from scratch -- e.g. read a directory of
configuration and write it back to the same directory.

**Note:** This principle requires symmetric inputs and outputs.

Dynamic logic may be written using templates or DSLs -- which would not support the read-write
workflow -- by merging the newly generated template / DSL output resources with the input
resources.

### Targets

Unlike the Kubernetes control-plane, which reads and writes from the apiserver, kpt
reads and writes from arbitrary sources, so long as they provide resource configuration.

- Local files
- Files stored in git
- Command stdin & stdout
- Apiserver endpoints
