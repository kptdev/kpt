---
title: "Publishing a blueprint"
linkTitle: "Blueprints"
weight: 5
type: docs
description: >
    Writing effective blueprint packages
---

{{% pageinfo color="warning" %}}
# Notice: Under Development
{{% /pageinfo %}}

*Reusable, customizable components can be built and shared as blueprint
packages.*

Blueprints are a p**attern for developing reusable configuration** as packages.
They incorporate  **best practices and policies** defined by an organization.

Blueprints may be used to accelerate on-boarding and increase configuration
quality. 

{{% pageinfo color="primary" %}}
Because packages can be updated, blueprint consumers can pull in the
latest best practices and policies as they are updated.
See the consumer update guide for more info.
{{% /pageinfo %}}

### Examples of blueprints

- Languages
  - Java / Node / Ruby / Python / Golang application
  
- Frameworks
  - Spring, Express, Rails, Django
  
- Platforms
  - Kubeflow, Spark
  
- Applications / Stacks
  - Rails Backend + Node Frontend + Prometheus
  - Spring Cloud Microservices (discovery-server, config-server,
    api-gateway, admin-server, hystrix, various backends) 

- Infrastructure
  - CloudSQL + Pubsub + GKE
  

This guide covers how to write effective blueprint packages with `kpt` 
and `kustomize`.

## Overview

Blueprint packages are developed 

Blueprint packages will typically be composed of **"publisher" owned pieces
and "consumer" owned pieces.**

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

