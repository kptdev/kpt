---
title: "Kustomize"
linkTitle: "Kustomize"
weight: 1
type: docs
description: >
    Publish kustomize bundles as packages of configuration 
---

*A kustomization.yaml is just another configuration file and works great for
breaking packages into pieces.*

Kustomize can be used to create packages with advanced structuring, instead of
just a single flat YAML bundle:

- Highlight pieces users are encouraged to modify by creating and documenting
  patches.
- Define non-consumer-editable pieces of the package (referenced as kustomize
  remote bases) as separate from consumer-editable pieces (included in the
  package).  
- Create a package that can be deployed to multiple environments by defining
  dev, staging, prod as separate kustomization.yaml directories within a single
  package.
- Enable cross-cutting edits by exposing namespace, common labels, etc in the
  kustomization.yaml

A kustomize package could simply be a kustomization.yaml referencing a base,
and a collection of patches for the consumer to edit.
