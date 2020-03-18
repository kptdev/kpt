---
title: "Publishing a blueprint"
linkTitle: "Blueprints"
weight: 5
type: docs
description: >
    Writing effective blueprint packages
---

*Reusable, customizable components can be built and shared as blueprint
packages.*

Blueprints are a **pattern for developing reusable configuration** as packages.
They incorporate  **best practices and policies** defined by an organization.

Blueprints may be used to accelerate on-boarding and increase configuration
quality. 

{{% pageinfo color="primary" %}}
Because packages can be updated, blueprint consumers can pull in the
latest best practices and policies as they are updated.
See the consumer update guide for more info.
{{% /pageinfo %}}

This guide covers how to write effective blueprint packages with `kpt` 
and `kustomize`.

### Examples of blueprints

#### Languages

Java / Node / Ruby / Python / Golang application

#### Frameworks

Spring, Express, Rails, Django

#### Platforms

Kubeflow, Spark
  
#### Applications / Stacks

Rails Backend + Node Frontend + Prometheus

Spring Cloud Microservices (discovery-server, config-server, api-gateway, 
admin-server, hystrix, various backends) 

#### Infrastructure Stacks

CloudSQL + Pubsub + GKE

## Overview

Blueprint packages are typically designed to be published and consumed by
different teams or individuals -- e.g. a 1 (publisher) to many (consumer)
model.

Blueprint packages may have consumer editable pieces (e.g. cpu, replicas)
and publisher editable pieces (e.g. health checks).  What is editable
and by whom will different between blueprint packages (e.g. the Java
blueprint would have the image as consumer editable, whereas the Spark
blueprint would have the image as publisher editable).

{{% pageinfo color="primary" %}}
While it is possible for consumers to override any publisher editable pieces
with kustomize patches, packages should be structured by publishers to
encourage or discourage editing various of pieces.
{{% /pageinfo %}}

## Factoring and merging configuration



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

