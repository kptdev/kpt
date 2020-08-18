---
title: "TypeScript Function SDK"
linkTitle: "TypeScript Function SDK"
weight: 5
type: docs
description: >
   Writing functions in TypeScript.
---

## TypeScript Function SDK

We provide an opinionated Typescript SDK for implementing config functions.
This provides various advantages:

- **General-purpose language:** Domain-Specific Languages begin their life
  with a reasonable feature set, but often grow over time. They bloat in order
  to accommodate the tremendous variety of customer use cases. Rather than
  follow this same course, config functions employ a true, general-purpose
  programming language that provides:
  - Proper abstractions and language features
  - A extensive ecosystem of tooling (e.g. IDE support)
  - A comprehensive catalog of well-supported libraries
  - Robust community support and detailed documentation
- **Type-safety:** Kubernetes configuration is typed, and its schema is
  defined using the OpenAPI spec. Typescript has a sophisticated type system
  that accommodates the complexity of Kubernetes resources. The SDK enables
  generating Typescript classes for core and CRD types, providing safe and
  easy interaction with Kubernetes objects.
- **Batteries-included:** The SDK provides a simple, powerful API for querying
  and manipulating configuration files. It provides the scaffolding required
  to develop, build, test, and publish functions, allowing you to focus on
  implementing your business-logic.

## Next Steps

- Try out the [Typescript Quickstart].
- Read the complete [Typescript Developer Guide].
- Learn how to [run functions].

[Typescript Quickstart]: quickstart/
[Typescript Developer Guide]: develop/
[run functions]: ../../../consumer/function/
