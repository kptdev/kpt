# Variant construction pattern

If you look at the config workflows, you will notice that creating variant
of a package is a very common and frequent operation, so reducing the steps
required to create a variant can have significant benefits for the
package consumers. In this guide, we will look at some techniques
that a package author can use to enable automatic variant construction of a package.

## Types of packages

kpt packages comes in two flavors:  `abstract package` and
`deployable package instance`. An `abstract` package is a reususable package that
is used to create deployable instances that can be deployed to a
kubernetes cluster. In programming language terms, you can think of an `abstract`
packages as the class and `deployable instance` as the instances of the class.
`deployable` instances of package are also referred to as `variant` of the package.

Figure below shows a `package catalog` on the left that has `abstract` packages
and `deployable instances` on the right. A good pattern is to keep the abstract
packages and instance packages in separate repos and typically
`deployable instances` repo will be setup to auto deploy to a kubernetes cluster
using gitops tools such as `config-sync`, `fluxcd`, `argocd`.

![variant constructor pkg repo diagram](/static/images/variant-constructor-pkg-repo-diagram.png)

Resources in an `abstract` package have placeholder values that needs to be
substituated with actual values to make them ready for deployment.
For example, name of the namespace resource below has `example` as a placeholder
value. This is a part of the `abstract package`.

```yaml
apiVersion:v1
kind: Namespace
metadata:
  name: example # <-- this is a placeholder value
```

## Customizing identity of resources

A kpt package contains kubernetes resources. Whenever you are creating a
variant of the package, first step is to ensure unique identity of the
resources in that variant. For example, if the abstract package contains a
`namespace` resource, then the variant package should contain namespace resource
corresponding to that variant.

In a kubernetes cluster, resources are identified by their group, version, kind,
namespace and name (aka GVKNN). If resource is cluster scoped, then the
`metadata.name` uniquely identifies the resource in a cluster. If the resource
is namespace scoped, then (namespace, name) together identifies the resource uniquely.

[kpt-function-catalog](https://catalog.kpt.dev) provides two function that helps
with customizing the identify of the resources:

1. [set-namespace](https://catalog.kpt.dev/set-namespace/v0.3/): sets the
   namespace for all resources in the package.
2. [ensure-name-substring](https://catalog.kpt.dev/ensure-name-substring/v0.2/):
   sets the name of the resources in the package.

So you can use the appropriate functions from the catalog or implement a custom
function to ensure unique identity of the resources.

## Customizing non-identifier fields of resources

Package can use other functions such as `set-labels`, `set-annotations`, `apply-replacements`
or custom functions to transform other fields of resources.

## Core mechanism

Enabling automatic variant construction involves two steps:

1. Use functions to customize identity or other fields of the resources
2. Generating inputs for the functions declarared in the package

Here is an example of `Kptfile` of a package that uses `set-namespace` and `apply-transform`
to enable customization.

```Kptfile
# Kptfile
...
pipeline:
  mutators:
    - image: set-namespace:v0.3.4
      configPath: ...
    - image: apply-transform:v0.1.0
      configPath: ...
...
```

Now let's talk about the input to the functions. In most cases, variant's name
(deployable package instance name) itself can be used to derive unique identity
of the resources in the variant. For example, if I create a variant of
`microservice` package, I will name the deployable package instance to
`user-service` or `order-service`. So if the package's name is available to the
functions, then they can use it to customize the name/namespace of the resources.
So, starting with `kpt v1.0.0-beta.15+` , kpt makes `package's name` available
in a local `ConfigMap` at a well-known path `package-context.yaml` that contains
the package instance name and is available to functions during
`kpt fn render` or `kpt fn eval`.

Examples of `package-context.yaml` for abstract and deployable instance, For
abstract package the value of `name` field is `example` and for deployable
instance it contains actual value of the package name.

```yaml
# package-context.yaml
# package context for an abstract package.
# This is automatically created on `kpt pkg init`.
apiVersion: v1
kind: ConfigMap
data:
  name: example
```

```yaml
# package-context.yaml
# package context for a deployable instance of a package.
# This is automatically populated during `kpt pkg get`.
apiVersion: v1
kind: ConfigMap
data:
  name: my-pkg-instance
```

kpt supports a way to create `deployable pkg instance` such that `package-context`
is automatically populated with the deployable instance's name.

```shell
$kpt pkg get <pkg-location> my-pkg-instance --for-deployment
```

Now, let's look at how to provide the input to the functions.

So if you are using `set-namespace` function in your package, then
`set-namespace` function supports reading input from `package-context.yaml`.
Here is an example:

```Kptfile
...
pipeline:
  mutators:
    - image: set-namespace:v0.3.4
      configPath: package-context.yaml
...
```

So by using `package-context.yaml` as input, `set-namespace` uses `example` value
for an `abstract` package and variant's name for a deployable instance. same pattern
can be applied to other functions also. For example, [`namespace provisioning`](https://github.com/GoogleContainerTools/kpt-samples/tree/main/basens)
package uses `apply-replacements` function to set the RoleBinding group
using the name of the package.

In some cases, the inputs needed to generate the variant will come from
some external system or environment. Those can be generated imperatively or
manually edited after the package is forked using `kpt pkg get`. Additional
customizations could also be made at that point.

So for a package consumer, creating a deployable instance involves the following:

```shell
# pick name of the deployable instance say `my-pkg-instance`
$ kpt pkg get <path-to-abstract-pkg> <my-pkg-instance> --for-deployement

$ kpt fn render <my-pkg-instance>

```

## See it in action

If you want to see `variant constructor pattern` in action for a real use-case,
check out [`namespace provisioning using kpt CLI guide`](/guides/namespace-provisioning-cli.md).

## Summary

So with the above pattern and workflow, you can how package publisher can
enable automatic customization of deployable instance of a package with minimal
input i.e. package instance name.

## Future plans

Currently `--for-deployment` steps invokes built-in function to generate
`package-context.yaml`. We would like to make it extensible for users to invoke their
custom function for deploy workflow.