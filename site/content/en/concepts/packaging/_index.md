---
title: "Packaging"
linkTitle: "Packaging"
weight: 4
type: docs
description: >
   Packaging goals and design decisions
---

The two primary sets of capabilities that are required to enable reuse are:

1. The ability to distribute/publish/share, compose, and update groups of
   configuration artifacts, commonly known as packages.
2. The ability to adapt them to your use cases, which we call customization.

In order to facilitate programmatic operations, kpt:
1. Relies upon git as the source of truth
2. Represents configuration as data, specifically Kubernetes resources
   serialized as YAML or JSON.

For compatibility with other arbitrary formats, kpt supports generating
resource configuration data from templates, configuration DSLs, and programs.
