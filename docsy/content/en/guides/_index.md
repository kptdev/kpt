---
title: "Guides"
linkTitle: "Guides"
weight: 20
type: docs
menu:
  main:
    weight: 2
---

Guides to get started using kpt.

### Day 1 Workflow

![day1 workflow][day1workflow]

### Day N Workflow

![dayN workflow][dayNworkflow]

#### Example imperative package workflow

1. [kpt pkg get](get.md) to get a package
2. [kpt cfg set](../cfg/set.md), [kpt fn run](../fn/run.md) or `vi` to modify configuration
3. `git add` && `git commit`
4. `kubectl apply` to a cluster:
5. [kpt pkg update](update.md) to pull in new changes
6. `kubectl apply` to a cluster

#### Example declarative package workflow

1. [kpt pkg init](init.md)
2. [kpt pkg sync set](sync-set.md) dev version of a package
3. [kpt pkg sync set](sync-set.md) prod version of a package
4. `git add` && `git commit`
5. `kubectl apply --context dev` apply to dev
6. `kubectl apply --context prod` apply to prod

[day1workflow]: /diagrams/day1workflow.jpg
[dayNworkflow]: /diagrams/dayNworkflow.jpg
