---
title: "Publishing a blueprint"
linkTitle: "Blueprints"
weight: 5
type: docs
description: >
    Writing effective blueprint packages
---

# *Under Development*

*Reusable, customizable components can be built and shared as blueprint
packages.*

Blueprint packages are developed to give teams within and organization
a way to quickly get started building a new application, service or
product offering -- incorporating the best practices and policies
of the organization.

Because packages can be updated, blueprint consumers can pull in the
latest best practices and policies at any time with `kpt pkg update`.

This guide covers how to write effective blueprint packages with `kpt` 
and `kustomize`.

## Bases

- Consumer editable vs non-editable

## Patches

- Making editable pieces more editable

## Setters and Substitutions

- Pieces edited by humans
- Pieces to be edited by automation
- Limitations

## Commands, Args and Environment Variables

- Merges and overrides

## Generating ConfigMaps

## Updates

- versioning
- merge friendly updates

## Example

