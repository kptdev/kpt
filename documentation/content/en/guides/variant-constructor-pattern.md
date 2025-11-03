# Variant construction pattern

If you look at the config workflows, you will notice that creating a variant
of a package is a very frequent operation, so reducing the steps
required to create a variant can have significant benefits for the
package consumers. In this guide, we will look at some techniques
that a package author can use to enable automatic variant construction of a package.

## Types of packages

kpt packages comes in two flavors:  `abstract package` and
`deployable instance`. An `abstract` package is a reususable package that
is used to create deployable instances that can be deployed to a
kubernetes cluster. In programming language terms, you can think of an `abstract`
packages as the class and `deployable instance` as the instances of the class.
`deployable` instances of package are also referred to as `variant` of the package.

The figure below shows a `package catalog` on the left that has `abstract` packages
and `deployable instances` on the right. A good pattern is to keep the abstract
packages and instance packages in separate repos and typically
`deployable instances` repo will be setup to auto deploy to a kubernetes cluster
using gitops tools such as `config-sync`, `fluxcd`, and `argocd`.

![variant constructor pkg repo diagram](/images/variant-constructor-pkg-repo-diagram.png)

Resources in an `abstract` package have placeholder values that should be
substituted with actual values to make them ready for deployment.
For example, the name of the namespace resource below has `example` as a placeholder
value. This is a part of the `abstract package`.

```yaml
apiVersion:v1
kind: Namespace
metadata:
  name: example # <-- this is a placeholder value
```

## Customizing identity of resources

A kpt package contains kubernetes resources. Whenever you are creating a
variant of the package, first step is to ensure the resources in that variant have
a unique identity. For example, if the abstract package contains a
`namespace` resource, then the variant package should contain a `namespace` resource
corresponding to that variant.

In a kubernetes cluster, resources are identified by their group, version, kind,
namespace and name (also referred to as GVKNN). If a resource is cluster scoped,
then the `metadata.name` uniquely identifies the resource in a cluster. If the resource
is namespace scoped, then (`metadata.namespace`, `metadata.name`) together identifies the
resource uniquely.

The [Function Catalog](https://catalog.kpt.dev/function-catalog/) provides two function that help
with customizing the identify of the resources:

1. [set-namespace](https://catalog.kpt.dev/function-catalog/set-namespace/v0.4/): sets the
   namespace for all resources in the package.
2. [ensure-name-substring](https://catalog.kpt.dev/function-catalog/ensure-name-substring/v0.2/):
   sets the name of the resources in the package.

You can use appropriate functions from the catalog or implement a custom
function to ensure unique identities for all resources.

## Customizing non-identifier fields of resources

Packages can use other functions such as
[set-labels](https://catalog.kpt.dev/function-catalog/set-labels/v0.2/),
[set-annotations](https://catalog.kpt.dev/function-catalog/set-annotations/v0.1/),
[apply-replacements](https://catalog.kpt.dev/function-catalog/apply-replacements/v0.1/),
or custom functions to transform other fields of resources.

## Core mechanism

Enabling automatic variant construction involves two steps:

1. Use functions to customize identity or other fields of the resources
2. Generating inputs for the functions declarared in the package

Here is an example of `Kptfile` of a package that uses
[set-namespace](https://catalog.kpt.dev/function-catalog/set-namespace/v0.4/) and a custom
 function called `apply-transform` to enable customization.

```Kptfile
# Kptfile
...
pipeline:
  mutators:
    - image: set-namespace:latest
      configPath: ...
    - image: apply-transform:v0.1.0
      configPath: ...
...
```

Now let's talk about the input to the functions. In most cases, variant's name
(deployable instance name) itself can be used to derive unique identity
of the resources in the variant. For example, if I create a variant of the
`microservice` package, I will name the deployable instance to
`user-service` or `order-service`. So if the package's name is available to the
functions, then they can use it to customize the name/namespace of the resources.
So kpt makes `package name` available in a `ConfigMap` at a well-known path
`package-context.yaml` in `data.name` field. The `package-context.yaml` is available
to functions during `kpt fn render|eval`.

Here are examples of `package-context.yaml` for abstract and deployable instance:

```yaml
# package-context.yaml
# package context for an abstract package.
# This is automatically created on `kpt pkg init`.
apiVersion: v1
kind: ConfigMap
data:
  name: example # <-- placeholder value
```

```yaml
# package-context.yaml
# package context for a deployable instance of a package.
# This is automatically populated during `kpt pkg get`.
apiVersion: v1
kind: ConfigMap
data:
  name: my-pkg-instance # <- deployable instance name
```

kpt supports a way to create a `deployable instance` such that `package-context.yaml`
is automatically populated with the `deployable instance`'s name.

```shell
$kpt pkg get <pkg-location> my-pkg-instance --for-deployment
```

Now, let's look at how to provide the input to the functions.

If you are using `set-namespace` function in your package, then
`set-namespace` function supports reading input from `package-context.yaml`.
Here is an example:

```Kptfile
...
pipeline:
  mutators:
    - image: set-namespace:latest
      configPath: package-context.yaml
...
```

By using `package-context.yaml` as input, `set-namespace` uses the value `example`
for an `abstract` package and variant's name for a deployable instance. The
same pattern can also be applied to other functions. For example, a
"namespace provisioning" package might use the `apply-replacements` function to set
the `RoleBinding` group using the name of the package.

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
check out the [`namespace provisioning using kpt CLI guide`](/guides/namespace-provisioning-cli.md).

## Summary

With the above pattern and workflow, you can see how a package publisher can
enable automatic customization of deployable instance of a package with minimal
input. They just provide the package instance name.

## Future plans

Currently `--for-deployment` steps invokes built-in function to generate
`package-context.yaml`. We would like to make it extensible for users to invoke their
custom function for deploy workflows.
