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

*Reusable, customizable components can be built and shared as blueprints.*

## Overview

Blueprints are a **pattern for developing reusable, customizable
configuration**.  Blueprints are typically published and consumed by
different teams.

{{% pageinfo color="primary" %}}
Because packages can be updated to new versions, blueprint consumers
can pull in changes to a blueprint after fetching it.
{{% /pageinfo %}}

### Example use cases for blueprints

- **Languages**: Java / Node / Ruby / Python / Golang application
- **Frameworks**: Spring, Express, Rails, Django
- **Platforms**: Kubeflow, Spark
- **Applications / Stacks**:
  - Rails Backend + Node Frontend + Prometheus
  - Spring Cloud Microservices (discovery-server, config-server, api-gateway, 
    admin-server, hystrix, various backends) 
- **Infrastructure Stacks**: CloudSQL + Pubsub + GKE

{{< svg src="images/blueprint" >}}

```sh
# Optional: copy the mysql-kustomize blueprint to follow along
kpt pkg get https://github.com/GoogleContainerTools/kpt.git/package-examples/mysql-kustomize mysql
```

## Factor / denormalize the configuration data

Structuring blueprints into separate publisher and consumer focused pieces
provides a clean UX for consumers to modifys the package.

Example: provide separate directories with pieces consumers are expected to
edit (replicas) vs publisher implementation (health check command).

As a package publisher, it is important to think about **where and how you
want to promote customization.**

We will use [kustomize] to structure the package:

1. **Factoring out a common field value**
   - Example: `namespace`, `commonLabels`, `commonAnnotations`
2. **Factoring a single resource into multiple files**
   - Example: `resources` + `patches`

{{% pageinfo color="info" %}}
Remote kustomize bases may be used to reference the publisher focused pieces
directly from a git repository rather than including them in the package.

One disadvantage of this approach is that it creates a dependency on the
remote package being accessible in order to push -- if you can't fetch the
remote package, then you can't push changes.
{{% /pageinfo %}}

Example package structure:

```sh
$ tree mysql/
mysql/
├── Kptfile
├── README.md
├── instance
│   ├── kustomization.yaml
│   ├── service.yaml
│   └── statefulset.yaml
└── upstream
    ├── kustomization.yaml
    ├── service.yaml
    └── statefulset.yaml
```

The `upstream` directory acts as a kustomize base to the `instance` directory.
Upstream contains things most **consumers are unlikely to modify** --
e.g. the image (for off the shelf software), health check endpoints, etc.

The `instance` directory contains patches with fields populated for things
most **consumers are expected to modify** -- e.g. namespace, cpu, memory,
user, password, etc.

{{% pageinfo color="warning" %}}
While the package is structured into publisher and consumer focused pieces,
it is still possible for the package consumer to modify (via direct edits)
or override (via patches) any part of the package.

**Factoring is for UX, not for enforcement of specific configuration values.**
{{% /pageinfo %}}

## Commands, Args and Environment Variables

How do you configure applications in a way that can be extended or overridden --
how can consumers of a package specify new args, flags, environment variables
or configuration files and merge those with those defined by the package
publisher?

### Notes

- Commands and Args are non-associative arrays so it is not possible to
  target specific elements -- any changes replace the entire list of elements.
- Commands and Args are separate fields that are concatenated
- Commands and Args can use values from environment variables
- Environment variables are associative arrays, so it is possible to target
  specific elements within the list to be overridden or added.
- Environment variables can be pulled from ConfigMaps and Secrets
- Kustomize merges ConfigMaps and Secrets per-key (deep merges of the
  values is not supported).
- ConfigMaps and Secrets can be read from apps via environment variables or
  volumes.

Flags and arguments may be factored into publisher and consumer focused pieces
by **specifying the `command` in the `upstream` base dir and the `args` in the 
`instance` dir**.  This allows consumers to set and add flags using `args`
without erasing those defined by the publisher in the `command`.

When **specifying values for arguments or flag values, it is best to use an
environment variable read from a generated ConfigMap.**  This enables overriding
the value using kustomize's generators.

Example: Enable setting `--skip-grant-tables` as a flag on mysql.

*`# {"$ref": ...` comments are setter references, defined in the next section.*

```yaml
# mysql/instance/statefulset.yaml
# Wire ConfigMap value from kustomization.yaml to
# an environment variable used by an arg
apiVersion: apps/v1
kind: StatefulSet
...
spec:
  template:
    metadata:
      labels:
        app: release-name-mysql
    spec:
      containers:
      - name: mysql
        ...
        args:
        - --skip-grant-tables=$(SKIP_GRANT_TABLES)
        ...
        env:
        - name: SKIP_GRANT_TABLES
          valueFrom:
            configMapKeyRef:
              name: mysql
              key: skip-grant-tables
```

```yaml
# mysql/instance/kustomization.yaml
# Changing the literal changes the StatefulSet behavior
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
- name: mysql
  behavior: merge
  literals:
  # for bootstrapping the root table grants -- set to false after bootstrapped
  - "skip-grant-tables=true" # {"$ref":"#/definitions/io.k8s.cli.substitutions.skip-grant-tables"}
```

### Generating ConfigMaps and Secrets

Kustomize supports generating ConfigMaps and Secrets from the
kustomization.yaml.

- Generated objects have a suffix applied so that the name is unique for
  the data.  This ensures a rollout of Deployments and StatefulSets occurs.
- Generated object may have their values overridden by downstream consumers.

Example Upstream:

```yaml
# mysql/upstream
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
- name: mysql
  literals:
    - skip-grant-tables=true
```

Example Instance:

```yaml
# mysql/instance
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
configMapGenerator:
- name: mysql
  behavior: merge
  literals:
  - "skip-grant-tables=true" # {"$ref":"#/definitions/io.k8s.cli.substitutions.skip-grant-tables"}
  - "mysql-user=" # {"$ref":"#/definitions/io.k8s.cli.substitutions.mysql-user"}
  - "mysql-database=" # {"$ref":"#/definitions/io.k8s.cli.substitutions.mysql-database"}
```

## Setters and Substitutions

It may be desirable to provide user friendly commands for customizing the
package rather than exclusively through text editors and sed:

- Setting a value in several different patches at once
- Setting common or required values -- e.g. the image name for a Java app
  blueprint
- Setting the image tag to match the digest of an image that was just build.
- Setting a value from the environment when the package is fetched the first
  time -- e.g. GCP project.

Setters and substitutions are a way to define user and automation friendly
commands for performing structured edits of a configuration.

Combined with the preceding techniques, setters or substitutions can be used
to modify generated ConfigMaps and patches in the `instance` dir.

See the [setter] and [substitution] guides for details.

```yaml
# mysql/instance/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
#
# namespace is the namespace the mysql instance is run in
namespace: "" # {"$ref":"#/definitions/io.k8s.cli.setters.namespace"}
configMapGenerator:
- name: mysql
  behavior: merge
  literals:
  # for bootstrapping the root table grants -- set to false after bootstrapped
  - "skip-grant-tables=true" # {"$ref":"#/definitions/io.k8s.cli.substitutions.skip-grant-tables"}
  - "mysql-user=" # {"$ref":"#/definitions/io.k8s.cli.substitutions.mysql-user"}
  - "mysql-database=" # {"$ref":"#/definitions/io.k8s.cli.substitutions.mysql-database"}
...
```

```yaml
# mysql/instance/statefulset.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      containers:
      - name: mysql
...
        ports:
        - name: mysql
          containerPort: 3306 # {"$ref":"#/definitions/io.k8s.cli.setters.port"}
        resources:
          requests:
            cpu: 100m # {"$ref":"#/definitions/io.k8s.cli.setters.cpu"}
            memory: 256Mi # {"$ref":"#/definitions/io.k8s.cli.setters.memory"}
```

```yaml
# mysql/instance/service.yaml
apiVersion: v1
kind: Service
...
spec:
  ports:
  - name: mysql
    port: 3306 # {"$ref":"#/definitions/io.k8s.cli.setters.port"}
    targetPort: mysql
```

## Updates

Individual directories may have their own package versions by prefixing the
version with the directory path -- e.g.
`package-examples/mysql-kustomize/v0.1.0`.

When publishing a new version of a package, publishers should think about how
their changes will be merged into existing packages.

Changing values in the instance package is not recommended, but adding them
may be ok -- changes to fields will overwrite user changes to those same
fields, whereas adds will only conflict if the user added the same field.

[kustomize]: https://github.com/kubernetes-sigs/kustomize
[setter]: ../setters
[substitution]: ../substitutions
