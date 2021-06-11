---
title: "kpt and kustomize"
linkTitle: "kustomize"
type: docs
---

## Overview
Some of kpt users have proficiency with [kustomize] or already have 
configuration that relies on kustomize.  The similarities and differences 
between the tools are coverd by the [FAQ].

In this solution we will go through a pattern where you can use kpt for 
packaging and applying the final resources to a cluster, but leverage 
kustomize overlays for hydration of your final configuration.

Let's take a look at a package that leverages kustomize hydration:

```shell

```




[FAQ]: /faq/
[kustomize]: https://kustomize.io
