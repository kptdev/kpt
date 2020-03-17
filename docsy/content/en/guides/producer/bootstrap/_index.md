---
title: "Bootstrapping"
linkTitle: "Bootstrapping"
weight: 7
type: docs
description: >
    Bootstrap a package with content generated or published from another source.
---

## Fetched from another package

Fetch another package and use it as your starting point (e.g.
[kubernetes-examples](https://github.com/kubernetes/examples))

```sh
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/helloworld-set@v0.3.0 helloworld
```

## Generated from cli tools

Generate configuration from commandline tools (e.g.
[kubectl](https://kubectl.docs.kubernetes.io/pages/imperative_porcelain/creating_resources.html))

```sh
kubectl create deployment nginx --image nginx -o yaml --dry-run > deploy.yaml
```

## Copied from examples or blog posts

Some examples may be published as blog posts without being published
as a package in git.  These can be copied directly from their source.

```sh
curl https://raw.githubusercontent.com/kubernetes/website/master/content/en/examples/controllers/nginx-deployment.yaml --output nginx/nginx-deployment.yaml
```

## Generated from DSLs or templates

Generate configuration from templates (e.g. [helm](https://helm.sh/))

```sh
helm fetch stable/mysql
helm template mysql-1.3.1.tgz --output-dir .
```

## Generated from code

Generate configuration from application source code
(e.g. [dekorate](https://github.com/dekorateio/dekorate))

```sh
mvn clean package
```
